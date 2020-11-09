package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
	"github.com/lambertjamesd/sfz2n64/convert"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage sfz2n64 input.sfz output_prefix")
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

		if outExt == ".inst" {
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

		if outExt == ".inst" {
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
	} else if ext == ".aifc" || ext == ".aiff" {
		file, err := os.Open(input)

		if err != nil {
			log.Fatal(err)
		}

		defer file.Close()

		aiff, err := aiff.Parse(file)

		if err != nil {
			log.Fatal(err)
		}

		out, err := os.OpenFile(input+".out.aiff", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

		aiff.Serialize(out)

		if err != nil {
			log.Fatal(err)
		}

		defer out.Close()

		if aiff.Compressed {
			log.Println("Compressed")
		} else {
			log.Println("Not Compressed")
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
