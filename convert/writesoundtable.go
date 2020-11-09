package convert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/wav"
)

type SoundEntry struct {
	sound     al64.ALSound
	soundData []byte
}

func wavToSoundEntry(filename string) (*SoundEntry, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	waveFile, err := wav.Parse(file)

	if err != nil {
		return nil, err
	}

	if waveFile.Header.Format != wav.FORMAT_PCM {
		return nil, errors.New(fmt.Sprintf("%s should be pcm", filename))
	}

	if waveFile.Header.NChannels != 1 {
		return nil, errors.New(fmt.Sprintf("%s should have 1 channel", filename))
	}

	if waveFile.Header.BitsPerSample != 16 {
		return nil, errors.New(fmt.Sprintf("%s should have 16 bits per sample", filename))
	}

	swapEndian(waveFile.Data)

	var result SoundEntry

	result.sound.Envelope = &al64.ALEnvelope{
		AttackTime:   0,
		DecayTime:    0,
		ReleaseTime:  0,
		AttackVolume: 127,
		DecayVolume:  127,
	}

	result.sound.Wavetable = &al64.ALWavetable{
		Base:     0,
		Len:      int32(len(waveFile.Data)),
		Type:     al64.AL_RAW16_WAVE,
		AdpcWave: al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
		RawWave:  al64.ALRAWWaveInfo{Loop: nil},
	}

	result.sound.SamplePan = 64
	result.sound.SampleVolume = 127
	result.soundData = waveFile.Data

	return &result, nil
}

func aiffToSoundEntry(filename string) (*SoundEntry, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	aiffFile, err := aiff.Parse(file)

	if err != nil {
		return nil, err
	}

	if aiffFile.Common.NumChannels != 1 {
		return nil, errors.New(fmt.Sprintf("%s should have 1 channel", filename))
	}

	if aiffFile.Common.SampleSize != 16 {
		return nil, errors.New(fmt.Sprintf("%s should have 16 bits per sample", filename))
	}

	var result SoundEntry

	result.sound.Envelope = &al64.ALEnvelope{
		AttackTime:   0,
		DecayTime:    0,
		ReleaseTime:  0,
		AttackVolume: 127,
		DecayVolume:  127,
	}

	if aiffFile.Compressed {
		// TODO
	} else {
		result.sound.Wavetable = &al64.ALWavetable{
			Base:     0,
			Len:      int32(len(aiffFile.SoundData.WaveformData)),
			Type:     al64.AL_RAW16_WAVE,
			AdpcWave: al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
			RawWave:  al64.ALRAWWaveInfo{Loop: nil},
		}
		// TODO loops
	}

	result.sound.SamplePan = 64
	result.sound.SampleVolume = 127
	result.soundData = aiffFile.SoundData.WaveformData

	return &result, nil
}

func fileToSoundEntry(filename string) (*SoundEntry, error) {
	var ext = filepath.Ext(filename)

	if ext == ".wav" {
		return wavToSoundEntry(filename)
	} else if ext == ".aiff" || ext == ".aifc" {
		return aiffToSoundEntry(filename)
	} else {
		return nil, errors.New("Unrecongized sound extension")
	}
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
		soundData.Sounds = append(soundData.Sounds, &sound.sound)
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
