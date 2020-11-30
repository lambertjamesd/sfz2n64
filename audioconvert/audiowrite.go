package audioconvert

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/wav"
)

func EnsureDirectory(filename string) error {
	var dir = filepath.Dir(filename)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = EnsureDirectory(dir)
		if err != nil {
			return nil
		}
		return os.Mkdir(dir, 0776)
	}

	return nil
}

func ConvertCodebookToAL64(codebook *adpcm.Codebook) *al64.ALADPCMBook {
	var result al64.ALADPCMBook

	result.Order = int32(codebook.Order)
	result.NPredictors = int32(len(codebook.Predictors))

	result.Book = make([]int16, 8*result.Order*result.NPredictors)

	var currIdx = 0

	for pred := int32(0); pred < result.NPredictors; pred = pred + 1 {
		for order := int32(0); order < result.Order; order = order + 1 {
			for idx := 0; idx < 8; idx = idx + 1 {
				var val = codebook.Predictors[pred].Table[idx][order]

				if val > 0x7fff {
					result.Book[currIdx] = 0x7fff
				} else if val < -0x8000 {
					result.Book[currIdx] = -0x8000
				} else {
					result.Book[currIdx] = int16(val)
				}

				currIdx = currIdx + 1
			}
		}
	}

	return &result
}

func ConvertCodebook(alType *al64.ALADPCMBook) *adpcm.Codebook {
	var result adpcm.Codebook

	result.Order = int(alType.Order)

	var inputIndex = 0

	for pred := int32(0); pred < alType.NPredictors; pred = pred + 1 {
		var predictor adpcm.Predictor

		for idx := 0; idx < 8; idx = idx + 1 {
			predictor.Table[idx] = make([]int32, result.Order+8)
		}

		for order := int32(0); order < alType.Order; order = order + 1 {
			for idx := 0; idx < 8; idx = idx + 1 {
				predictor.Table[idx][order] = int32(alType.Book[inputIndex])
				inputIndex = inputIndex + 1
			}
		}

		adpcm.ExpandPredictor(&predictor, result.Order)

		result.Predictors = append(result.Predictors, predictor)
	}

	return &result
}

func convertLoop(alType *al64.ALADPCMloop) *adpcm.Loop {
	if alType == nil {
		return nil
	} else {
		return &adpcm.Loop{
			Start: int(alType.Start),
			End:   int(alType.End),
			Count: int(alType.Count),
			State: alType.State,
		}
	}
}

func EncodeSamples(data []int16, order binary.ByteOrder) []byte {
	var buffer bytes.Buffer

	for _, val := range data {
		binary.Write(&buffer, order, &val)
	}

	return buffer.Bytes()
}

func DecodeSamples(data []byte, order binary.ByteOrder) []int16 {
	var buffer = bytes.NewBuffer(data)
	var result = make([]int16, len(data)/2)

	for index, _ := range result {
		binary.Read(buffer, order, &result[index])
	}

	return result
}

func SwapEndian(data []byte) {
	for i := 0; i < len(data); i = i + 2 {
		data[i], data[i+1] = data[i+1], data[i]
	}
}

func WriteWav(filename string, wave *al64.ALWavetable, data []byte, sampleRate uint32) error {
	var waveFile wav.Wave

	if wave.Type == al64.AL_ADPCM_WAVE {
		var sampleCount = adpcm.NumberSamples(wave.Len)
		var frames = adpcm.DecodeADPCM(&adpcm.ADPCMEncodedData{
			NSamples:   int(sampleCount),
			SampleRate: float64(sampleRate),
			Codebook:   ConvertCodebook(wave.AdpcWave.Book),
			Loop:       convertLoop(wave.AdpcWave.Loop),
			Frames:     adpcm.ReadFrames(data),
		})

		data = EncodeSamples(frames.Samples, binary.LittleEndian)
		wave.Type = al64.AL_RAW16_WAVE
	} else {
		SwapEndian(data)
	}

	waveFile.Header.Format = wav.FORMAT_PCM
	waveFile.Header.NChannels = 1
	waveFile.Header.SampleRate = sampleRate
	waveFile.Header.ByteRate = sampleRate * 2
	waveFile.Header.BlockAlign = 2
	waveFile.Header.BitsPerSample = 16

	waveFile.Data = data

	EnsureDirectory(filename)

	waveFileOut, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer waveFileOut.Close()

	waveFile.Serialize(waveFileOut)

	return nil
}

func WriteAiff(filename string, wave *al64.ALWavetable, data []byte, sampleRate uint32) error {
	var aiffFile aiff.Aiff

	if wave.Type == al64.AL_ADPCM_WAVE {
		var sampleCount = adpcm.NumberSamples(wave.Len)
		var frames = adpcm.DecodeADPCM(&adpcm.ADPCMEncodedData{
			NSamples:   int(sampleCount),
			SampleRate: float64(sampleRate),
			Codebook:   ConvertCodebook(wave.AdpcWave.Book),
			Loop:       convertLoop(wave.AdpcWave.Loop),
			Frames:     adpcm.ReadFrames(data),
		})

		data = EncodeSamples(frames.Samples, binary.BigEndian)
		wave.Type = al64.AL_RAW16_WAVE
	}

	aiffFile.Compressed = false

	aiffFile.Common = &aiff.CommonChunk{
		NumChannels:     1,
		NumSampleFrames: wave.Len / 2,
		SampleSize:      16,
		SampleRate:      aiff.ExtendedFromF64(float64(sampleRate)),
		CompressionType: 0,
		CompressionName: "",
	}

	if wave.RawWave.Loop != nil {
		aiffFile.Markers = &aiff.MarkerChunk{
			Markers: []aiff.Marker{aiff.Marker{
				ID:       1,
				Position: wave.RawWave.Loop.Start,
				Name:     "start",
			}, aiff.Marker{
				ID:       2,
				Position: wave.RawWave.Loop.End,
				Name:     "end",
			},
			}}

		aiffFile.Instrument = &aiff.InstrumentChunk{
			BaseNote:     0,
			Detune:       0,
			LowNote:      0,
			HighNote:     0,
			LowVelocity:  0,
			HighVelocity: 0,
			Gain:         0,
			SustainLoop:  aiff.Loop{PlayMode: 1, BeginLoop: 1, EndLoop: 2},
			ReleaseLoop:  aiff.Loop{PlayMode: 0, BeginLoop: 0, EndLoop: 0},
		}
	}

	aiffFile.SoundData = &aiff.SoundDataChunk{
		Offset:       0,
		BlockSize:    0,
		WaveformData: data,
	}

	EnsureDirectory(filename)

	aiffFileOut, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer aiffFileOut.Close()

	aiffFile.Serialize(aiffFileOut)

	return nil
}

func WriteAifc(filename string, wave *al64.ALWavetable, data []byte, sampleRate uint32) error {
	var aiffFile aiff.Aiff

	if wave.Type == al64.AL_RAW16_WAVE {
		// TODO encode
	}

	aiffFile.Compressed = true

	aiffFile.Common = &aiff.CommonChunk{
		NumChannels:     1,
		NumSampleFrames: adpcm.NumberSamples(wave.Len),
		SampleSize:      16,
		SampleRate:      aiff.ExtendedFromF64(float64(sampleRate)),
		CompressionType: 0x56415043,
		CompressionName: "VADPCM ~4-1",
	}

	var codesBuffer bytes.Buffer

	var len uint8 = 0xB
	binary.Write(&codesBuffer, binary.BigEndian, &len)
	codesBuffer.WriteString("VADPCMCODES")

	var version uint16 = 1
	binary.Write(&codesBuffer, binary.BigEndian, &version)

	version = uint16(wave.AdpcWave.Book.Order)
	binary.Write(&codesBuffer, binary.BigEndian, &version)
	version = uint16(wave.AdpcWave.Book.NPredictors)
	binary.Write(&codesBuffer, binary.BigEndian, &version)

	for _, val := range wave.AdpcWave.Book.Book {
		binary.Write(&codesBuffer, binary.BigEndian, &val)
	}

	aiffFile.Application = append(aiffFile.Application, &aiff.ApplicationChunk{
		Signature: 0x73746F63,
		Data:      codesBuffer.Bytes(),
	})

	if wave.AdpcWave.Loop != nil {
		var loopBuffer bytes.Buffer

		var len uint8 = 0xB
		binary.Write(&loopBuffer, binary.BigEndian, &len)
		loopBuffer.WriteString("VADPCMLOOPS")

		var version uint16 = 1
		binary.Write(&loopBuffer, binary.BigEndian, &version)
		// num loops
		binary.Write(&loopBuffer, binary.BigEndian, &version)

		binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.Start)
		binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.End)
		binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.Count)

		for _, val := range wave.AdpcWave.Loop.State {
			binary.Write(&loopBuffer, binary.BigEndian, &val)
		}

		aiffFile.Application = append(aiffFile.Application, &aiff.ApplicationChunk{
			Signature: 0x73746F63,
			Data:      loopBuffer.Bytes(),
		})
	}

	aiffFile.SoundData = &aiff.SoundDataChunk{
		Offset:       0,
		BlockSize:    0,
		WaveformData: data,
	}

	EnsureDirectory(filename)

	aiffFileOut, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer aiffFileOut.Close()

	aiffFile.Serialize(aiffFileOut)

	return nil
}
