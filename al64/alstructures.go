package al64

type ALWaveType uint8

const (
	AL_ADPCM_WAVE ALWaveType = 0
	AL_RAW16_WAVE ALWaveType = 1
)

const ADPCMFSIZE = 16
const BANK_REVISION = 0x4231

type ADPCM_STATE [ADPCMFSIZE]int16

type ALADPCMBook struct {
	Order       int32
	NPredictors int32
	Book        []int16 /* Must be 8-byte aligned */
}

type ALADPCMloop struct {
	Start uint32
	End   uint32
	Count uint32
	State ADPCM_STATE
}

type ALRawLoop struct {
	Start uint32
	End   uint32
	Count uint32
}

type ALADPCMWaveInfo struct {
	Loop *ALADPCMloop
	Book *ALADPCMBook
}

type ALRAWWaveInfo struct {
	Loop *ALRawLoop
}

type ALWavetable struct {
	Base int32
	Len  int32
	Type ALWaveType
	// Flags    uint8
	AdpcWave ALADPCMWaveInfo
	RawWave  ALRAWWaveInfo
	// this parameter is not actually part of the n64 data structure
	// but is used to make it easier to pass ALWavetable with it's
	// cooresponding data
	DataFromTable  []byte
	FileSampleRate uint32
}

type ALKeyMap struct {
	VelocityMin uint8
	VelocityMax uint8
	KeyMin      uint8
	KeyMax      uint8
	KeyBase     uint8
	Detune      uint8
}

type ALEnvelope struct {
	AttackTime   int32
	DecayTime    int32
	ReleaseTime  int32
	AttackVolume uint8
	DecayVolume  uint8
}

type ALSound struct {
	Envelope     *ALEnvelope
	KeyMap       *ALKeyMap
	Wavetable    *ALWavetable
	SamplePan    uint8
	SampleVolume uint8
	// Flags        uint8
}

type ALInstrument struct {
	Volume   uint8
	Pan      uint8
	Priority uint8
	// Flags     uint8
	TremType  uint8
	TremRate  uint8
	TremDepth uint8
	TremDelay uint8
	VibType   uint8
	VibRate   uint8
	VibDepth  uint8
	VibDelay  uint8
	BendRange int16
	// SoundCount int16
	SoundArray []*ALSound
}

type ALBank struct {
	// InstCount  int16
	// Flags      uint8
	// Pad        uint8
	SampleRate uint32
	Percussion *ALInstrument
	InstArray  []*ALInstrument
}

type ALBankFile struct {
	// Revision int16
	// BankCount int16
	BankArray []*ALBank
}

func TblFromBank(bankFile *ALBankFile) []byte {
	var result []byte = nil
	var base int32 = 0

	for _, bank := range bankFile.BankArray {
		for _, inst := range bank.InstArray {
			if inst != nil {
				for _, sound := range inst.SoundArray {
					if sound != nil && sound.Wavetable != nil {
						var padding = ((base + 0xf) & ^0xf) - base

						if padding != 0 {
							base = base + padding
							result = append(result, make([]byte, padding)...)
						}

						sound.Wavetable.Base = base
						sound.Wavetable.Len = int32(len(sound.Wavetable.DataFromTable))
						base += sound.Wavetable.Len

						result = append(result, sound.Wavetable.DataFromTable...)
					}
				}
			}
		}
	}

	return result
}

func (inst *ALInstrument) CorrectOverlap() {
	if inst == nil {
		return
	}

	for upperIndex := 0; upperIndex < len(inst.SoundArray); upperIndex++ {
		for lowerIndex := 0; lowerIndex < upperIndex; lowerIndex++ {
			var a = inst.SoundArray[upperIndex].KeyMap
			var b = inst.SoundArray[lowerIndex].KeyMap

			if a.KeyMax >= b.KeyMin && a.KeyMax <= b.KeyMax {
				a.KeyMax = b.KeyMin - 1
			}

			if a.KeyMin >= b.KeyMin && a.KeyMin <= b.KeyMin {
				a.KeyMin = b.KeyMax + 1
			}

			if b.KeyMax >= a.KeyMin && b.KeyMax <= a.KeyMax {
				b.KeyMax = a.KeyMin - 1
			}

			if b.KeyMin >= a.KeyMin && b.KeyMin <= a.KeyMin {
				b.KeyMin = a.KeyMax + 1
			}
		}
	}
}

func (bankFile *ALBankFile) CorrectOverlap() {
	for _, bank := range bankFile.BankArray {
		// bank.Percussion.CorrectOverlap()
		for _, inst := range bank.InstArray {
			inst.CorrectOverlap()
		}
	}
}
