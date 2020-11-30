package audioconvert

import (
	"encoding/binary"

	"github.com/lambertjamesd/sfz2n64/al64"
)

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

		return (int16)(float32(currentSample)*(1-lerpValue) + float32(nextSample)*lerpValue)
	}
}

func Resample(input []int16, from int, to int) []int16 {
	var result []int16 = make([]int16, len(input)*to/from)

	var scale = float32(from) / float32(to)

	for index, _ := range result {
		result[index] = GetSample(input, float32(index)*scale)
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
	var resampled = Resample(samples, from, to)

	if wavetable.RawWave.Loop != nil {
		var loop al64.ALRawLoop

		loop.Start = uint32(int(wavetable.RawWave.Loop.Start) * to / from)
		loop.End = uint32(int(wavetable.RawWave.Loop.End) * to / from)
		loop.Count = wavetable.RawWave.Loop.Count

		result.RawWave.Loop = &loop
	}

	result.Base = wavetable.Base
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

	result.AttackTime = int32(int(envelope.AttackTime) * to / from)
	result.DecayTime = int32(int(envelope.DecayTime) * to / from)
	result.ReleaseTime = int32(int(envelope.ReleaseTime) * to / from)
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
