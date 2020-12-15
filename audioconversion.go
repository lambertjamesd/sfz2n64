package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
)

func convertAudio(input string, output string, compressionSettings *adpcm.CompressionSettings) {
	sound, err := audioconvert.ReadWavetable(input)

	if err != nil {
		log.Fatal(err)
	}

	var outExt = filepath.Ext(output)

	if outExt == ".table" {
		var codebook *adpcm.Codebook = nil
		if sound.Wavetable.Type == al64.AL_RAW16_WAVE {
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
}
