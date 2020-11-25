package convert

import (
	"os"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
)

func WriteSoundBank(outputName string, inputSounds []string) error {
	var sounds []*al64.ALSound

	for _, input := range inputSounds {
		sound, err := audioconvert.ReadWavetable(input)

		if err != nil {
			return err
		}

		sounds = append(sounds, sound)
	}

	var offset int32 = 0
	var combinedData []byte = nil
	var soundData al64.SoundArray = al64.SoundArray{Sounds: nil}

	for _, sound := range sounds {
		var padding = ((offset + 0xf) & ^0xf) - offset

		if padding != 0 {
			offset = offset + padding
			combinedData = append(combinedData, make([]byte, padding)...)
		}

		sound.Wavetable.Base = sound.Wavetable.Base + offset
		offset = offset + int32(len(sound.Wavetable.DataFromTable))
		soundData.Sounds = append(soundData.Sounds, sound)
		combinedData = append(combinedData, sound.Wavetable.DataFromTable...)
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
