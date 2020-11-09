package convert

import (
	"os"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
)

type SoundEntry struct {
	sound     *al64.ALSound
	soundData []byte
}

func fileToSoundEntry(filename string) (*SoundEntry, error) {
	result, soundData, err := audioconvert.ReadWavetable(filename)

	if err != nil {
		return nil, err
	}

	return &SoundEntry{
		result,
		soundData,
	}, nil
}

func WriteSoundBank(outputName string, inputSounds []string) error {
	var sounds []*SoundEntry

	for _, input := range inputSounds {
		sound, err := fileToSoundEntry(input)

		if err != nil {
			return err
		}

		sounds = append(sounds, sound)
	}

	var offset int32 = 0
	var combinedData []byte = nil
	var soundData al64.SoundArray = al64.SoundArray{Sounds: nil}

	for _, sound := range sounds {
		sound.sound.Wavetable.Base = sound.sound.Wavetable.Base + offset
		offset = offset + int32(len(sound.soundData))
		soundData.Sounds = append(soundData.Sounds, sound.sound)
		combinedData = append(combinedData, sound.soundData...)
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
