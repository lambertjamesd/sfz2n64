package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage sfz2n64 input.sfz output_prefix")
	}

	var input = os.Args[1]

	var ext = filepath.Ext(input)

	if ext == ".sfz" {
		sfzFile, err := sfz.ParseSfz(input)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Parsed %s with %d sections", input, len(sfzFile.Sections))
	} else if ext == ".ctl" {
		file, err := os.Open(input)

		if err != nil {
			log.Fatal(err)
		}

		bankFile, err := al64.ReadBankFile(file)

		if err != nil {
			log.Fatal(err)
		}

		output, err := os.OpenFile(input+".recomp", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

		if err != nil {
			log.Fatal(err)
		}

		err = bankFile.Serialize(output)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Parsted tbl %s with %d banks", input, len(bankFile.BankArray))
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
	} else {
		log.Fatal(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file", input))
	}
}
