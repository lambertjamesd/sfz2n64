package audioconvert

import (
	"encoding/binary"
	"math"

	"github.com/lambertjamesd/sfz2n64/al64"
)

func ConvertSampleLocation(location int, to int, from int) int {
	var result = float64(location)*float64(to)/float64(from) + 0.5
	return int(math.Floor(result))
}

func lerpSample(a int16, b int16, lerp float32) int16 {
	return (int16)(float32(a)*(1-lerp) + float32(b)*lerp)
}

func GetSample(input []int16, at float32) int16 {
	var asInt = int(at)

	if asInt < 0 {
		return input[0]
	} else if asInt+1 >= len(input) {
		return input[len(input)-1]
	} else {
		var currentSample = input[asInt]
		var nextSample = input[asInt+1]

		var lerpValue = at - float32(asInt)

		return lerpSample(currentSample, nextSample, lerpValue)
	}
}

func Resample(input []int16, from int, to int) []int16 {
	var result []int16 = make([]int16, ConvertSampleLocation(len(input), to, from))

	var scale = float32(from) / float32(to)

	for index, _ := range result {
		result[index] = GetSample(input, float32(index)*scale)
	}

	return result
}

func ResampleLooped(input []int16, from int, to int, loopStart int, loopEnd int) []int16 {
	var result []int16 = make([]int16, ConvertSampleLocation(len(input), to, from))
	var scale = float32(from) / float32(to)

	var convertedStart = ConvertSampleLocation(loopStart, to, from)
	var convertedEnd = ConvertSampleLocation(loopEnd, to, from)

	// ensure that the first sample in the loop is accurate
	var scaleOffset = float32(loopStart) - float32(convertedStart)*scale

	for index := 0; index < convertedEnd && index < len(result); index++ {
		result[index] = GetSample(input, float32(index)*scale+scaleOffset)
	}

	scaleOffset = float32(loopEnd) - float32(convertedEnd)*scale

	for index := convertedStart; index < convertedEnd && index < len(result); index++ {
		var lerp = float32(index-convertedStart) / float32(convertedEnd-1-convertedStart)

		var inputSample = GetSample(input, float32(index)*scale+scaleOffset)
		result[index] = lerpSample(result[index], inputSample, lerp)
	}

	for index := convertedEnd; index < len(result); index++ {
		result[index] = GetSample(input, float32(index)*scale+scaleOffset)
	}

	return result
}

func ResampleWavetable(wavetable *al64.ALWavetable, to int, from int) *al64.ALWavetable {
	if wavetable == nil {
		return nil
	} else if wavetable.Type != al64.AL_RAW16_WAVE {
		// TODO resample compressed data
		return wavetable
	}

	var result al64.ALWavetable

	if wavetable.FileSampleRate != 0 {
		from = int(wavetable.FileSampleRate)
	}

	var samples = DecodeSamples(wavetable.DataFromTable, binary.BigEndian)
	var resampled []int16

	if wavetable.RawWave.Loop != nil {
		var loop al64.ALRawLoop

		loop.Start = uint32(ConvertSampleLocation(int(wavetable.RawWave.Loop.Start), to, from))
		loop.End = uint32(ConvertSampleLocation(int(wavetable.RawWave.Loop.End), to, from))
		loop.Count = wavetable.RawWave.Loop.Count

		resampled = ResampleLooped(samples, from, to, int(wavetable.RawWave.Loop.Start), int(wavetable.RawWave.Loop.End))

		result.RawWave.Loop = &loop
	} else {
		resampled = Resample(samples, from, to)
	}

	result.Base = 0
	result.Len = int32(2 * len(resampled))
	result.Type = wavetable.Type
	result.DataFromTable = EncodeSamples(resampled, binary.BigEndian)
	result.FileSampleRate = uint32(to)

	return &result
}

func ResampleEnvelope(envelope *al64.ALEnvelope, to int, from int) *al64.ALEnvelope {
	if envelope == nil {
		return nil
	}

	var result al64.ALEnvelope

	result.AttackTime = int32(ConvertSampleLocation(int(envelope.AttackTime), to, from))
	result.DecayTime = int32(ConvertSampleLocation(int(envelope.DecayTime), to, from))
	result.ReleaseTime = int32(ConvertSampleLocation(int(envelope.ReleaseTime), to, from))
	result.AttackVolume = envelope.AttackVolume
	result.DecayVolume = envelope.DecayVolume

	return &result
}

func ResampleSound(sound *al64.ALSound, to int, from int) *al64.ALSound {
	var result al64.ALSound

	result.Envelope = ResampleEnvelope(sound.Envelope, to, from)
	result.KeyMap = sound.KeyMap
	result.Wavetable = ResampleWavetable(sound.Wavetable, to, from)
	result.SamplePan = sound.SamplePan
	result.SampleVolume = sound.SampleVolume

	return &result
}

func ResampleInstrument(instrument *al64.ALInstrument, to int, from int) *al64.ALInstrument {
	if instrument == nil {
		return nil
	}

	var result al64.ALInstrument

	result.Volume = instrument.Volume
	result.Pan = instrument.Pan
	result.Priority = instrument.Priority
	result.TremType = instrument.TremType
	result.TremRate = instrument.TremRate
	result.TremDepth = instrument.TremDepth
	result.TremDelay = instrument.TremDelay
	result.VibType = instrument.VibType
	result.VibRate = instrument.VibRate
	result.VibDepth = instrument.VibDepth
	result.VibDelay = instrument.VibDelay
	result.BendRange = instrument.BendRange

	for _, sound := range instrument.SoundArray {
		result.SoundArray = append(result.SoundArray, ResampleSound(sound, to, from))
	}

	return &result
}

func ResampleBank(bank *al64.ALBank, to int) *al64.ALBank {
	var result al64.ALBank

	result.SampleRate = uint32(to)
	result.Percussion = ResampleInstrument(bank.Percussion, to, int(bank.SampleRate))

	for _, instrument := range bank.InstArray {
		result.InstArray = append(result.InstArray, ResampleInstrument(instrument, to, int(bank.SampleRate)))
	}

	return &result
}

func ResampleBankFile(bankfile *al64.ALBankFile, to int) *al64.ALBankFile {
	var result al64.ALBankFile

	for _, input := range bankfile.BankArray {
		result.BankArray = append(result.BankArray, ResampleBank(input, to))
	}

	return &result
}
