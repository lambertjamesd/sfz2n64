package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/convert"
)

type SFZConvertArgs struct {
	TargetSampleRate    int
	BankSequenceMapping string
}

func ParseBankConvertArgs(args map[string]interface{}) (*SFZConvertArgs, error) {
	var result SFZConvertArgs

	intermediate, _ := args["--sample-rate"]
	sampleRate, _ := intermediate.(int64)
	result.TargetSampleRate = int(sampleRate)

	intermediate, _ = args["--bank_sequence_mapping"]
	bankSequenceMapping, _ := intermediate.(string)
	result.BankSequenceMapping = bankSequenceMapping

	return &result, nil
}

func ParseCompressionSettings(args map[string]interface{}) (*adpcm.CompressionSettings, error) {
	var result adpcm.CompressionSettings = adpcm.DefaultCompressionSettings()

	intermediate, _ := args["--order"]
	order, _ := intermediate.(int64)
	result.Order = int(order)

	intermediate, _ = args["--frame-size"]
	frameSize, _ := intermediate.(int64)
	result.FrameSize = int(frameSize)

	intermediate, _ = args["--threshold"]
	threshold, _ := intermediate.(float64)
	result.Threshold = threshold

	intermediate, _ = args["--bits"]
	bits, _ := intermediate.(int64)
	result.Bits = int(bits)

	intermediate, _ = args["--refine-iterations"]
	refineIters, _ := intermediate.(int64)
	result.RefineIters = int(refineIters)

	return &result, nil
}

func main() {
	var args Args = NewArgs("sfz2n64 [options] -o output.sfz|output.ins|output.ctl input.sfz|input.ins|input.ctl")

	args.AddFlagArg([]string{"-h", "--help"}, "print this help message")
	args.AddStringArg([]string{"-o", "--output"}, "the output file", "")
	args.AddIntegerArg([]string{"--sample-rate"}, "changes the sample rate of instrument banks", 0, 0, 200000)
	args.AddStringArg([]string{"--bank_sequence_mapping"}, "A list of midi files used to filter out unused sounds and instruments", "")

	args.AddIntegerArg([]string{"--order"}, "the order used in adpcm compression", 2, 1, 16)
	args.AddIntegerArg([]string{"--frame-size"}, "the number of samples to include in a single adpcm frame", 16, 16, 16)
	args.AddFloatArg([]string{"--threshold"}, "the threshold used in adpcm compression", 10, 1, 32)
	args.AddIntegerArg([]string{"--bits"}, "the number of bits to use for adpcm compression", 2, 1, 4)
	args.AddIntegerArg([]string{"--refine-iterations"}, "the number of refinement iterations to use in adpcm compression", 2, 1, 20000)
	args.AddFlagArg([]string{"--compress"}, "compress any uncompressed audio when converting")

	namedArgs, orderedArgs, errors := args.Parse(os.Args[1:len(os.Args)])

	intermediate, _ := namedArgs["--help"]
	showHelp, _ := intermediate.(bool)

	intermediate, _ = namedArgs["--output"]
	output, _ := intermediate.(string)

	if showHelp || len(errors) > 0 || len(output) == 0 || len(orderedArgs) == 0 {
		for _, err := range errors {
			fmt.Println(err.Error())
		}

		fmt.Println(args.CreateHelpMessage())
		return
	}

	var input = orderedArgs[0]

	var ext = filepath.Ext(input)
	var outExt = filepath.Ext(output)

	if isRomFile(ext) && isBankFile(outExt) {
		extractFromRom(input, output)
	} else if isRomFile(ext) && outExt == ".mid" || outExt == ".midi" {
		extractMidiFromRom(input, output)
	} else if isBankFile(ext) && isBankFile(outExt) {
		args, err := ParseBankConvertArgs(namedArgs)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		convertBank(input, output, args)
	} else if outExt == ".sounds" {
		intermediate, _ = namedArgs["--compress"]
		shouldCompress, _ := intermediate.(bool)

		var compressionSettings *adpcm.CompressionSettings

		if shouldCompress {
			var err error
			compressionSettings, err = ParseCompressionSettings(namedArgs)

			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		err := convert.WriteSoundBank(output, orderedArgs, compressionSettings)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Wrote sound array to " + output)
	} else if ext == ".aifc" || ext == ".aiff" || ext == ".wav" || ext == ".aif" {
		compressionSettings, err := ParseCompressionSettings(namedArgs)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		convertAudio(input, output, compressionSettings)
	} else if ext == ".mid" && isBankFile(outExt) {
		extractMidi(input, output)
	} else {
		fmt.Println(fmt.Sprintf("Invalid input file '%s'. Expected .sfz or .ctl file\n", input))
		os.Exit(1)
	}
}
