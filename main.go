package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
	"github.com/lambertjamesd/sfz2n64/convert"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func ParseArgs(args []string) (map[string]string, []string) {
	var named = make(map[string]string)
	var ordered []string = nil

	var lastName string = ""

	for _, curr := range args {
		if lastName != "" {
			named[lastName] = curr
			lastName = ""
		} else if len(curr) != 0 && curr[0] == '-' {
			lastName = curr
		} else {
			ordered = append(ordered, curr)
		}
	}

	return named, ordered
}

type SFZConvertArgs struct {
	TargetSampleRate int
}

func ParseSFZConvertArgs(args map[string]string) (*SFZConvertArgs, error) {
	var result SFZConvertArgs

	for name, value := range args {
		if name == "-s" || name == "--sample" {
			sampleRate, err := strconv.ParseInt(value, 10, 32)

			if err != nil {
				return nil, err
			}

			result.TargetSampleRate = int(sampleRate)
		} else {
			return nil, errors.New("Unrecognized input " + name)
		}
	}

	return &result, nil
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal(`Usage
	sfz2n64 input.sfz output.ins [-s --sample_rate sampleRate]
	sfz2n64 input.sfz output.ctl [-s --sample_rate sampleRate]

	sfz2n64 input.ctl output.ins
	sfz2n64 input.ctl output.sfz

	sfz2n64 input.ins output.sfz
	sfz2n64 input.ins output.ctl
`)
	}

	namedArgs, orderedArgs := ParseArgs(os.Args)

	var input = orderedArgs[1]

	var ext = filepath.Ext(input)
	var output = orderedArgs[2]

	if ext == ".sfz" {
		args, err := ParseSFZConvertArgs(namedArgs)

		if err != nil {
			log.Fatal(err)
		}

		sfzFile, err := sfz.ParseSfz(input)

		if err != nil {
			log.Fatal(err)
		}

		bankFile, err := convert.Sfz2N64(sfzFile, input)

		if err != nil {
			log.Fatal(err)
		}

		if args.TargetSampleRate != 0 {
			bankFile = audioconvert.ResampleBankFile(bankFile, args.TargetSampleRate)
		}

		var outExt = filepath.Ext(output)

		var tblData = audioconvert.BuildTbl(bankFile)

		var isSingle = convert.SfzIsSingleInstrument(sfzFile)

		if outExt == ".ins" {
			var instrumentNames []string = nil

			if isSingle {
				var instName = filepath.Base(input)
				var ext = filepath.Ext(instName)

				instrumentNames = append(instrumentNames, instName[0:len(instName)-len(ext)])
			}

			err = convert.WriteInsFile(bankFile, tblData, output, instrumentNames, isSingle)
		} else if outExt == ".ctl" {
			err = convert.WriteSfzFile(bankFile, tblData, output)
		} else {
			log.Fatal("Outut file should be of type .ins or .sfz")
		}

		fmt.Printf("Wrote instrument file to %s", output)
	} else if ext == ".ctl" {
		file, err := os.Open(input)

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		bankFile, err := al64.ReadBankFile(file)

		if err != nil {
			log.Fatal(err)
		}

		tblData, err := ioutil.ReadFile(input[0:len(input)-4] + ".tbl")

		if err != nil {
			log.Fatal(err)
		}

		var outExt = filepath.Ext(output)

		if outExt == ".ins" {
			err = convert.WriteInsFile(bankFile, tblData, output, nil, false)
		} else if outExt == ".sfz" {
			err = convert.WriteSfzFile(bankFile, tblData, output)
		} else {
			log.Fatal("Outut file should be of type .ins or .sfz")
		}

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Wrote instrument file to %s", output)
	} else if ext == ".ins" {
		file, err := ioutil.ReadFile(input)

		if err != nil {
			log.Fatal(err)
		}

		insFile, parseErrors := al64.ParseIns(string(file), input, func(waveFilename string) (*al64.ALWavetable, error) {
			sound, err := audioconvert.ReadWavetable(waveFilename)

			if err != nil {
				return nil, err
			}

			return sound.Wavetable, nil
		})

		if len(parseErrors) != 0 {
			for _, err := range parseErrors {
				log.Println(err.Error())
			}
		} else {
			var outExt = filepath.Ext(output)

			if outExt == ".sfz" {
				err = convert.WriteSfzFile(insFile.BankFile, insFile.TblData, output)
			} else if outExt == ".ctl" {
				err = convert.WriteSfzFile(insFile.BankFile, insFile.TblData, output)
			} else {
				log.Fatal("Outut file should be of type .sfz or .ctl")
			}

			if err != nil {
				log.Fatal(err)
			}

			log.Printf("Number of elements parsed %d", len(insFile.StructureOrder))
		}
	} else if ext == ".aifc" || ext == ".aiff" || ext == ".wav" || ext == ".aif" {
		sound, err := audioconvert.ReadWavetable(input)

		if err != nil {
			log.Fatal(err)
		}

		var outExt = filepath.Ext(output)

		if outExt == ".table" {
			var codebook *adpcm.Codebook = nil
			if sound.Wavetable.Type == al64.AL_RAW16_WAVE {
				var compressionSettings = adpcm.DefaultCompressionSettings()
				codebook, err = adpcm.CalculateCodebook(
					audioconvert.DecodeSamples(sound.Wavetable.DataFromTable, binary.BigEndian),
					&compressionSettings,
				)

				if err != nil {
					log.Fatal(err)
				}
			} else {
				codebook = audioconvert.ConvertCodebook(sound.Wavetable.AdpcWave.Book)
			}

			outputFile, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

			if err != nil {
				log.Fatal(err)
			}

			codebook.Serialize(outputFile)

			fmt.Printf("Wrote table to %s", output)
		} else {
			fmt.Printf("Could not convert %s to %s", input, output)
		}
	} else if ext == ".sounds" {
		err := convert.WriteSoundBank(input, orderedArgs[2:len(os.Args)])

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Wrote sound array to " + input)
	} else {
		log.Fatal(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file", input))
	}
}
