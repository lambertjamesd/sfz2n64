package main

import (
	"encoding/binary"
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

	sampleRateString, ok := args["--sample-rate"]

	if ok {
		sampleRate, err := strconv.ParseInt(sampleRateString, 10, 32)

		if err != nil {
			return nil, err
		}

		result.TargetSampleRate = int(sampleRate)
		delete(args, "--sample-rate")
	}

	return &result, nil
}

func ParseCompressionSettings(args map[string]string) (*adpcm.CompressionSettings, error) {
	var result adpcm.CompressionSettings = adpcm.DefaultCompressionSettings()

	orderString, ok := args["--order"]

	if ok {
		order, err := strconv.ParseInt(orderString, 10, 32)

		if err != nil {
			return nil, err
		}

		result.Order = int(order)
		delete(args, "--order")
	}

	frameSizeString, ok := args["--frame-size"]

	if ok {
		frameSize, err := strconv.ParseInt(frameSizeString, 10, 32)

		if err != nil {
			return nil, err
		}

		result.FrameSize = int(frameSize)
		delete(args, "--frame-size")
	}

	thresholdString, ok := args["--threshold"]

	if ok {
		threshold, err := strconv.ParseFloat(thresholdString, 64)

		if err != nil {
			return nil, err
		}

		result.Threshold = threshold
		delete(args, "--threshold")
	}

	bitsString, ok := args["--bits"]

	if ok {
		bits, err := strconv.ParseInt(bitsString, 10, 32)

		if err != nil {
			return nil, err
		}

		result.Bits = int(bits)
		delete(args, "--bits")
	}

	refineItersString, ok := args["--refine-iterations"]

	if ok {
		refineIters, err := strconv.ParseInt(refineItersString, 10, 32)

		if err != nil {
			return nil, err
		}

		result.RefineIters = int(refineIters)
		delete(args, "--refine-iterations")
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
			err = convert.WriteCtlFile(output, bankFile)
		} else {
			log.Fatal("Outut file should be of type .ins or .sfz\n")
		}

		fmt.Printf("Wrote instrument file to %s\n", output)
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
			log.Fatal("Outut file should be of type .ins or .sfz\n")
		}

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Wrote instrument file to %s\n", output)
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
				err = convert.WriteCtlFile(output, insFile.BankFile)
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
				compressionSettings, err := ParseCompressionSettings(namedArgs)

				if err != nil {
					log.Fatal(err)
				}

				codebook, err = adpcm.CalculateCodebook(
					audioconvert.DecodeSamples(sound.Wavetable.DataFromTable, binary.BigEndian),
					compressionSettings,
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
		} else if outExt == ".aifc" {
			compressionSettings, err := ParseCompressionSettings(namedArgs)

			if err != nil {
				log.Fatal(err)
			}

			err = audioconvert.CompressWithSettings(sound.Wavetable, input, compressionSettings)

			if err != nil {
				log.Fatal(err)
			}

			err = audioconvert.WriteAifc(output, sound.Wavetable, sound.Wavetable.DataFromTable, sound.Wavetable.FileSampleRate)

			if err != nil {
				log.Fatal(err)
			}
		} else if outExt == ".aif" || outExt == ".aiff" {
			err = audioconvert.WriteAiff(output, sound.Wavetable, sound.Wavetable.DataFromTable, sound.Wavetable.FileSampleRate)

			if err != nil {
				log.Fatal(err)
			}
		} else if outExt == ".wav" {
			err = audioconvert.WriteWav(output, sound.Wavetable, sound.Wavetable.DataFromTable, sound.Wavetable.FileSampleRate)
		} else {
			fmt.Printf("Could not convert %s to %s\n", input, output)
		}
	} else if ext == ".sounds" {
		shouldCompress, _ := namedArgs["--compress"]

		var compressionSettings *adpcm.CompressionSettings

		if shouldCompress == "true" {
			var err error
			compressionSettings, err = ParseCompressionSettings(namedArgs)

			if err != nil {
				log.Fatal(err)
			}
		}

		err := convert.WriteSoundBank(input, orderedArgs[2:len(orderedArgs)], compressionSettings)

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Wrote sound array to " + input)
	} else {
		log.Fatal(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file", input))
	}
}
