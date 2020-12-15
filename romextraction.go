package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/convert"
	"github.com/lambertjamesd/sfz2n64/midi"
	"github.com/lambertjamesd/sfz2n64/romextractor"
)

func extractMidiFromRom(input string, output string) {
	var outExt = filepath.Ext(output)

	data, err := ioutil.ReadFile(input)

	if err != nil {
		log.Fatal(err)
	}

	romextractor.CorrectByteswap(data)

	songs := romextractor.FindMidi(data)

	for index, song := range songs {
		var withoutExt = output[0 : len(output)-len(outExt)]
		var newFile = fmt.Sprintf("%s_%d.mid", withoutExt, index)

		outFile, err := os.OpenFile(newFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

		if err != nil {
			log.Fatal(err)
		}

		midi.WriteMidi(outFile, song)
	}

	log.Println(fmt.Sprintf("Found %d songs", len(songs)))

}

func extractFromRom(input string, output string) {
	var outExt = filepath.Ext(output)

	data, err := ioutil.ReadFile(input)

	if err != nil {
		log.Fatal(err)
	}

	romextractor.CorrectByteswap(data)

	banks := romextractor.FindBanks(data)

	if err != nil {
		log.Fatal(err)
	}

	var finalBanks []*al64.ALBankWithTable = nil

	for _, bank := range banks {
		tblOffset, tblLen, err := romextractor.FindTbl(bank, data)

		if err == nil {
			var tblData = data[tblOffset : tblOffset+tblLen]
			al64.WriteTlbIntoBank(bank, tblData)
			finalBanks = append(finalBanks, &al64.ALBankWithTable{
				Bank: bank,
				Tbl:  tblData,
			})
		} else {
			log.Println("Failed to find tbl data for bank")
		}
	}

	for index, bank := range finalBanks {
		var withoutExt = output[0 : len(output)-len(outExt)]
		var newDir = fmt.Sprintf("%s_%d", withoutExt, index)

		dirState, err := os.Stat(newDir)

		if os.IsNotExist(err) {
			err = os.Mkdir(newDir, 0777)
			if err != nil {
				log.Fatal(err)
			}
		} else if !dirState.IsDir() {
			log.Fatal(fmt.Sprintf("%s is not a directory", newDir))
		}

		var finalPath = filepath.Join(newDir, filepath.Base(withoutExt)+outExt)

		err = writeBank(input, finalPath, bank.Bank, bank.Tbl, false)

		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("Wrote instrument file to %s\n", output)
		}
	}

	log.Println(fmt.Sprintf("Found %d banks", len(finalBanks)))
}

func extractMidi(input string, output string) {
	midFile, err := os.Open(input)

	if err != nil {
		log.Fatal(err)
	}

	defer midFile.Close()

	midi, err := midi.ReadMidi(midFile)

	if err != nil {
		log.Fatal(err)
	}

	bankFile, _, _, err := parseInputBank(output)

	if err != nil {
		log.Fatal(err)
	}

	var maxActiveNotes = convert.SimplifyMidi(midi, bankFile.BankArray[0], 20)

	log.Println(fmt.Sprintf("Max number of active notes %d\n", maxActiveNotes))
}
