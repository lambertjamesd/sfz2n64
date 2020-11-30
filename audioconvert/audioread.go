package audioconvert

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/wav"
)

func wavToSoundEntry(filename string) (*al64.ALSound, error) {
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

	SwapEndian(waveFile.Data)

	var result al64.ALSound

	result.Envelope = &al64.ALEnvelope{
		AttackTime:   0,
		DecayTime:    int32(1000000 * len(waveFile.Data) / 2 / int(waveFile.Header.SampleRate)),
		ReleaseTime:  0,
		AttackVolume: 127,
		DecayVolume:  127,
	}

	result.Wavetable = &al64.ALWavetable{
		Base:           0,
		Len:            int32(len(waveFile.Data)),
		Type:           al64.AL_RAW16_WAVE,
		AdpcWave:       al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
		RawWave:        al64.ALRAWWaveInfo{Loop: nil},
		DataFromTable:  waveFile.Data,
		FileSampleRate: uint32(waveFile.Header.SampleRate),
	}

	result.Wavetable.DataFromTable = waveFile.Data
	result.Wavetable.FileSampleRate = waveFile.Header.SampleRate

	return &result, nil
}

func aiffToSoundEntry(filename string) (*al64.ALSound, error) {
	file, err := os.Open(filename)

	if err != nil {
		return nil, err
	}

	defer file.Close()

	aiffFile, err := aiff.Parse(file)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing file: %s error: %s", filename, err.Error()))
	}

	if aiffFile.Common.NumChannels != 1 {
		return nil, errors.New(fmt.Sprintf("%s should have 1 channel", filename))
	}

	if aiffFile.Common.SampleSize != 16 {
		return nil, errors.New(fmt.Sprintf("%s should have 16 bits per sample", filename))
	}

	var result al64.ALSound

	var sampleRate = uint32(aiff.F64FromExtended(aiffFile.Common.SampleRate))

	if aiffFile.Compressed {
		result.Wavetable = &al64.ALWavetable{
			Base:           0,
			Len:            int32(len(aiffFile.SoundData.WaveformData)),
			Type:           al64.AL_ADPCM_WAVE,
			AdpcWave:       al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
			RawWave:        al64.ALRAWWaveInfo{Loop: nil},
			DataFromTable:  aiffFile.SoundData.WaveformData,
			FileSampleRate: sampleRate,
		}

		result.Envelope = &al64.ALEnvelope{
			AttackTime:   0,
			DecayTime:    int32(1000000 * len(aiffFile.SoundData.WaveformData) * 16 / 9 / int(sampleRate)),
			ReleaseTime:  0,
			AttackVolume: 127,
			DecayVolume:  127,
		}

		for _, chunk := range aiffFile.Application {
			if chunk.Signature == 0x73746F63 {
				var buffer = bytes.NewBuffer(chunk.Data)
				var headerLen uint8
				binary.Read(buffer, &binary.BigEndian, &headerLen)
				var data = make([]byte, headerLen)
				buffer.Read(data)

				var headerName = string(data)

				if headerName == "VADPCMCODES" {
					var version uint16
					binary.Read(buffer, binary.BigEndian, &version)

					if version == 1 {
						var book al64.ALADPCMBook
						binary.Read(buffer, binary.BigEndian, &version)
						book.Order = int32(version)
						binary.Read(buffer, binary.BigEndian, &version)
						book.NPredictors = int32(version)

						book.Book = make([]int16, 8*book.Order*book.NPredictors)

						for i := 0; i < len(book.Book); i++ {
							binary.Read(buffer, binary.BigEndian, &book.Book[i])
						}
						result.Wavetable.AdpcWave.Book = &book
					}
				} else if headerName == "VADPCMLOOPS" {
					var version uint16
					var loops uint16
					binary.Read(buffer, binary.BigEndian, &version)
					binary.Read(buffer, binary.BigEndian, &loops)

					if version == 1 && loops == 1 {
						var loop al64.ALADPCMloop
						binary.Read(buffer, binary.BigEndian, &loop.Start)
						binary.Read(buffer, binary.BigEndian, &loop.End)
						binary.Read(buffer, binary.BigEndian, &loop.Count)

						for index, _ := range loop.State {
							binary.Read(buffer, binary.BigEndian, &loop.State[index])
						}
						result.Wavetable.AdpcWave.Loop = &loop
					}
				}
			}
		}

		if result.Wavetable.AdpcWave.Book == nil {
			return nil, errors.New("Could not find book in wavetable")
		}
	} else {
		result.Wavetable = &al64.ALWavetable{
			Base:           0,
			Len:            int32(len(aiffFile.SoundData.WaveformData)),
			Type:           al64.AL_RAW16_WAVE,
			AdpcWave:       al64.ALADPCMWaveInfo{Loop: nil, Book: nil},
			RawWave:        al64.ALRAWWaveInfo{Loop: nil},
			DataFromTable:  aiffFile.SoundData.WaveformData,
			FileSampleRate: sampleRate,
		}

		result.Envelope = &al64.ALEnvelope{
			AttackTime:   0,
			DecayTime:    int32(1000000 * len(aiffFile.SoundData.WaveformData) / 2 / int(sampleRate)),
			ReleaseTime:  0,
			AttackVolume: 127,
			DecayVolume:  127,
		}

		if aiffFile.Instrument != nil && aiffFile.Markers != nil {
			var loop al64.ALRawLoop
			var start = aiffFile.Markers.FindMarker(aiffFile.Instrument.SustainLoop.BeginLoop)
			var end = aiffFile.Markers.FindMarker(aiffFile.Instrument.SustainLoop.EndLoop)

			if start == nil {
				start = aiffFile.Markers.FindMarker(aiffFile.Instrument.ReleaseLoop.BeginLoop)
				end = aiffFile.Markers.FindMarker(aiffFile.Instrument.ReleaseLoop.EndLoop)
			}

			if start != nil {
				loop.Start = start.Position
			}

			if end != nil {
				loop.End = end.Position
			}

			loop.Count = 0xffffffff

			result.Wavetable.RawWave.Loop = &loop
		}
		// TODO loops
	}

	return &result, nil
}

func insToSoundEntry(filename string) (*al64.ALSound, error) {
	file, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	instFile, parseErrors := al64.ParseIns(string(file), filename, func(waveFilename string) (*al64.ALWavetable, error) {
		sound, err := ReadWavetable(waveFilename)

		if err != nil {
			return nil, err
		}

		return sound.Wavetable, nil
	})

	if len(parseErrors) > 0 {
		return nil, parseErrors[0]
	}

	sound, ok := instFile.StructureByName["Sound"]

	if !ok {
		return nil, errors.New(fmt.Sprintf("%s should have a single sound object named Sound", filename))
	}

	asSound, ok := sound.(*al64.ALSound)

	if !ok {
		return nil, errors.New(fmt.Sprintf("%s should have a single sound object named Sound", filename))
	}

	if asSound.Envelope == nil {
		asSound.Envelope = &al64.ALEnvelope{
			AttackTime:   0,
			DecayTime:    int32(1000000 * len(asSound.Wavetable.DataFromTable) / 2 / int(asSound.Wavetable.FileSampleRate)),
			ReleaseTime:  0,
			AttackVolume: 127,
			DecayVolume:  127,
		}
	}

	return asSound, nil
}

func ReadWavetable(filename string) (*al64.ALSound, error) {
	var ext = filepath.Ext(filename)

	if ext == ".wav" {
		return wavToSoundEntry(filename)
	} else if ext == ".aiff" || ext == ".aifc" || ext == ".aif" {
		return aiffToSoundEntry(filename)
	} else if ext == ".ins" {
		return insToSoundEntry(filename)
	} else {
		return nil, errors.New("Not a supported sound file " + filename)
	}
}

func buildTblInstrument(instrument *al64.ALInstrument, result []byte) []byte {
	for _, sound := range instrument.SoundArray {
		sound.Wavetable.Base = int32(len(result))
		result = append(result, sound.Wavetable.DataFromTable...)
	}

	return result
}

func BuildTbl(banks *al64.ALBankFile) []byte {
	var result []byte = nil

	for _, bank := range banks.BankArray {
		if bank.Percussion != nil {
			result = buildTblInstrument(bank.Percussion, result)
		}

		for _, instrument := range bank.InstArray {
			if instrument != nil {
				result = buildTblInstrument(instrument, result)
			}
		}
	}

	return result
}
