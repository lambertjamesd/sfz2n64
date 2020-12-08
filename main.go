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
	"github.com/lambertjamesd/sfz2n64/midi"
	"github.com/lambertjamesd/sfz2n64/romextractor"
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
	TargetSampleRate    int
	BankSequenceMapping string
}

func ParseBankConvertArgs(args map[string]string) (*SFZConvertArgs, error) {
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

	bankSequenceMapping, ok := args["--bank_sequence_mapping"]

	if ok {
		result.BankSequenceMapping = bankSequenceMapping
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

func main() {
	if len(os.Args) < 3 {
		log.Fatal(`Usage
	sfz2n64 input.sfz|input.ins|input.ctl output.sfz|output.ins|output.ctl [--sample_rate sampleRate] [--bank_sequence_mapping sequences.txt]
`)
	}

	namedArgs, orderedArgs := ParseArgs(os.Args)

	var input = orderedArgs[1]

	var ext = filepath.Ext(input)
	var output = orderedArgs[2]
	var outExt = filepath.Ext(output)

	if isRomFile(ext) && isBankFile(outExt) {
		extractFromRom(input, output)
	} else if isRomFile(ext) && outExt == ".mid" || outExt == ".midi" {
		extractMidiFromRom(input, output)
	} else if isBankFile(ext) && isBankFile(outExt) {
		args, err := ParseBankConvertArgs(namedArgs)

		if err != nil {
			log.Fatal(err)
		}

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
				bankFile.BankArray[i] = convert.RemoveUnusedSounds(bankFile.BankArray[i], bankMapping[i])
			}
		}

		if args.TargetSampleRate != 0 {
			bankFile = audioconvert.ResampleBankFile(bankFile, args.TargetSampleRate)
		}

		err = writeBank(input, output, bankFile, tblData, isSingleInstrument)

		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("Wrote instrument file to %s\n", output)
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
	} else if ext == ".mid" && isBankFile(outExt) {
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

	} else {
		log.Fatal(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file\n", input))
	}
}
