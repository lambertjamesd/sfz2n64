package al64

type ALWaveType uint8

const (
	AL_ADPCM_WAVE ALWaveType = 0
	AL_RAW16_WAVE ALWaveType = 1
)

const ADPCMFSIZE = 16

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
	AdpcWave *ALADPCMWaveInfo
	RawWave  *ALRAWWaveInfo
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
	AttackVolume int16
	DecayVolume  int16
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
	Pad        uint8
	SampleRate uint32
	Percussion *ALInstrument
	InstArray  []*ALInstrument
}

type ALBankFile struct {
	Revision int16
	// BankCount int16
	BankArray []*ALBank
}
