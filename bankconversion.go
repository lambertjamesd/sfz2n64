package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
	"github.com/lambertjamesd/sfz2n64/convert"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func isBankFile(ext string) bool {
	return ext == ".sfz" || ext == ".ctl" || ext == ".ins"
}

func isRomFile(ext string) bool {
	return ext == ".n64" || ext == ".z64" || ext == ".v64"
}

func parseInputBank(input string) (*al64.ALBankFile, []byte, bool, error) {
	var ext = filepath.Ext(input)

	var bankFile *al64.ALBankFile
	var tblData []byte = nil
	var isSingleInstrument = false

	if ext == ".sfz" {
		sfzFile, err := sfz.ParseSfz(input)

		isSingleInstrument = convert.SfzIsSingleInstrument(sfzFile)

		if err != nil {
			return nil, nil, false, err
		}

		bankFile, err = convert.Sfz2N64(sfzFile, input)

		if err != nil {
			return nil, nil, false, err
		}

		tblData = audioconvert.BuildTbl(bankFile)
	} else if ext == ".ctl" {
		file, err := os.Open(input)

		if err != nil {
			return nil, nil, false, err
		}

		defer file.Close()

		bankFile, err = al64.ReadBankFile(file)

		if err != nil {
			return nil, nil, false, err
		}

		tblData, err = ioutil.ReadFile(input[0:len(input)-4] + ".tbl")

		if err != nil {
			return nil, nil, false, err
		}
	} else if ext == ".ins" {
		file, err := ioutil.ReadFile(input)

		if err != nil {
			return nil, nil, false, err
		}

		instFile, parseErrors := al64.ParseIns(string(file), input, func(waveFilename string) (*al64.ALWavetable, error) {
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
			return nil, nil, false, errors.New("Could not parse ins file\n")
		}

		bankFile = instFile.BankFile
		tblData = instFile.TblData
	} else {
		return nil, nil, false, errors.New("Could not handle input file type")
	}

	return bankFile, tblData, isSingleInstrument, nil
}

func writeBank(input string, output string, bankFile *al64.ALBankFile, tblData []byte, isSingleInstrument bool) error {
	var outExt = filepath.Ext(output)

	if outExt == ".sfz" {
		return convert.WriteSfzFile(bankFile, tblData, output)
	} else if outExt == ".ctl" {
		return convert.WriteCtlFile(output, bankFile)
	} else if outExt == ".ins" {
		var instrumentNames []string = nil

		if isSingleInstrument {
			var instName = filepath.Base(input)
			var ext = filepath.Ext(instName)

			instrumentNames = append(instrumentNames, instName[0:len(instName)-len(ext)])
		}

		return convert.WriteInsFile(bankFile, tblData, output, instrumentNames, isSingleInstrument)
	} else {
		return errors.New("Could not write file")
	}
}

func convertBank(input string, output string, args *SFZConvertArgs) {
	bankFile, tblData, isSingleInstrument, err := parseInputBank(input)

	if err != nil {
		log.Fatal(err)
	}

	if args.BankSequenceMapping != "" {
		bankMapping, err := convert.ParseBankUsageFile(args.BankSequenceMapping)

		if err != nil {
			log.Fatal(err)
		}

		for i := 0; i < len(bankMapping) && i < len(bankFile.BankArray); i++ {
			bank, err := convert.RemoveUnusedSounds(bankFile.BankArray[i], bankMapping[i])

			if err != nil {
				log.Fatal(err)
			} else {
				bankFile.BankArray[i] = bank
			}
		}
	}

	if args.TargetSampleRate != 0 {
		bankFile = audioconvert.ResampleBankFile(bankFile, args.TargetSampleRate)
		tblData = bankFile.LayoutTbl(nil)
	}

	err = writeBank(input, output, bankFile, tblData, isSingleInstrument)

	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Printf("Wrote instrument file to %s\n", output)
	}
}
