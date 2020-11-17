package main

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
	"github.com/lambertjamesd/sfz2n64/convert"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal(`Usage
	sfz2n64 input.sfz output.inst
	sfz2n64 input.sfz output.ctl

	sfz2n64 input.ctl output.inst
	sfz2n64 input.ctl output.sfz
`)
	}

	var input = os.Args[1]

	var ext = filepath.Ext(input)
	var output = os.Args[2]

	if ext == ".sfz" {
		sfzFile, err := sfz.ParseSfz(input)

		if err != nil {
			log.Fatal(err)
		}

		bankFile, err := convert.Sfz2N64(sfzFile, input)

		if err != nil {
			log.Fatal(err)
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
			log.Fatal("Outut file should be of type .inst or .sfz")
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
			log.Fatal("Outut file should be of type .inst or .sfz")
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

		_, parseErrors := al64.ParseIns(string(file), input)

		if len(parseErrors) != 0 {
			for _, err := range parseErrors {
				log.Println(err.FormatError())
			}
		} else {

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
		err := convert.WriteSoundBank(input, os.Args[2:len(os.Args)])

		if err != nil {
			log.Fatal(err)
		}

		log.Println("Wrote sound array to " + input)
	} else {
		log.Fatal(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file", input))
	}
}
