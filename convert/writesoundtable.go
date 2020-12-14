package convert

import (
	"os"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
)

func WriteSoundBank(outputName string, inputSounds []string, compressionSettings *adpcm.CompressionSettings) error {
	var sounds []*al64.ALSound

	for _, input := range inputSounds {
		sound, err := audioconvert.ReadWavetable(input)

		if err != nil {
			return err
		}

		if compressionSettings != nil && sound.Wavetable.Type == al64.AL_RAW16_WAVE {
			err = audioconvert.CompressWithSettings(sound.Wavetable, input, compressionSettings)

			if err != nil {
				return err
			}
		}

		sounds = append(sounds, sound)
	}

	var combinedData []byte = nil
	var soundData al64.SoundArray = al64.SoundArray{Sounds: nil}

	for _, sound := range sounds {
		combinedData = sound.LayoutTbl(combinedData)

		soundData.Sounds = append(soundData.Sounds, sound)
	}

	outFile, err := os.OpenFile(outputName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer outFile.Close()

	err = soundData.Serialize(outFile)

	if err != nil {
		return err
	}

	tblFile, err := os.OpenFile(outputName+".tbl", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer tblFile.Close()

	_, err = tblFile.Write(combinedData)

	if err != nil {
		return err
	}

	return nil
}

func WriteCtlFile(outputName string, bankFile *al64.ALBankFile) error {
	tblData := bankFile.LayoutTbl(nil)

	outFile, err := os.OpenFile(outputName, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer outFile.Close()

	err = bankFile.Serialize(outFile)

	if err != nil {
		return err
	}

	tblFile, err := os.OpenFile(outputName+".tbl", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer tblFile.Close()

	_, err = tblFile.Write(tblData)

	if err != nil {
		return err
	}

	return nil
}
