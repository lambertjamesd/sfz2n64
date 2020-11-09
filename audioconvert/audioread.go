package audioconvert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/wav"
)

func wavToSoundEntry(filename string) (*al64.ALSound, []byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, nil, err
	}

	defer file.Close()

	waveFile, err := wav.Parse(file)

	if err != nil {
		return nil, nil, err
	}

	if waveFile.Header.Format != wav.FORMAT_PCM {
		return nil, nil, errors.New(fmt.Sprintf("%s should be pcm", filename))
	}

	if waveFile.Header.NChannels != 1 {
		return nil, nil, errors.New(fmt.Sprintf("%s should have 1 channel", filename))
	}

	if waveFile.Header.BitsPerSample != 16 {
		return nil, nil, errors.New(fmt.Sprintf("%s should have 16 bits per sample", filename))
	}

	SwapEndian(waveFile.Data)

	var result al64.ALSound

	result.Envelope = &al64.ALEnvelope{
		AttackTime:   0,
		DecayTime:    0,
		ReleaseTime:  0,
		AttackVolume: 127,
		DecayVolume:  127,
	}

	result.Wavetable = &al64.ALWavetable{
		Base:     0,
		Len:      int32(len(waveFile.Data)),
		Type:     al64.AL_RAW16_WAVE,
		AdpcWave: al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
		RawWave:  al64.ALRAWWaveInfo{Loop: nil},
	}

	return &result, waveFile.Data, nil
}

func aiffToSoundEntry(filename string) (*al64.ALSound, []byte, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, nil, err
	}

	defer file.Close()

	aiffFile, err := aiff.Parse(file)

	if err != nil {
		return nil, nil, err
	}

	if aiffFile.Common.NumChannels != 1 {
		return nil, nil, errors.New(fmt.Sprintf("%s should have 1 channel", filename))
	}

	if aiffFile.Common.SampleSize != 16 {
		return nil, nil, errors.New(fmt.Sprintf("%s should have 16 bits per sample", filename))
	}

	var result al64.ALSound

	if aiffFile.Compressed {
		// TODO
	} else {
		result.Wavetable = &al64.ALWavetable{
			Base:     0,
			Len:      int32(len(aiffFile.SoundData.WaveformData)),
			Type:     al64.AL_RAW16_WAVE,
			AdpcWave: al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
			RawWave:  al64.ALRAWWaveInfo{Loop: nil},
		}
		// TODO loops
	}

	return &result, aiffFile.SoundData.WaveformData, nil
}

func ReadWavetable(filename string) (*al64.ALSound, []byte, error) {
	var ext = filepath.Ext(filename)

	if ext == ".wav" {
		return wavToSoundEntry(filename)
	} else if ext == ".aiff" || ext == ".aifc" {
		return aiffToSoundEntry(filename)
	} else {
		return nil, nil, errors.New("Not a supported sound file " + filename)
	}

}
