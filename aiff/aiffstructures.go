package aiff

type ExtendedFloat struct {
	Sign bool
	Exponent uint16
	Mantissa uint64
}

type CommonChunk struct {
	NumChannels int16
	NumSampleFrames int32
	SampleSize int16
	SampleRate ExtendedFloat
}

type Marker struct {
	ID int16
	Position int32
	Name String
}

type MarkerChunk {
	Markers []Marker
}

type Loop {
	PlayMode int16
	BeginLoop int16
	EndLoop int16
}

type InstrumentChunk struct {
	BaseNote uint8
	Detune uint8
	LowNote uint8
	HighNote uint8
	LowVelocity uint8
	HighVelocity uint8
	Gain int16
	SustainLoop Loop
	ReleaseLoop Loop
}

type ApplicationChunk struct {
	Signature string
	Data []byte
}

type SoundDataChunk struct {
	Offset uint32
	BlockSize uint32
	WaveformData []uint8
}

type Aiff struct {
	Common *CommonChunk
	SoundData *SoundDataChunk
	Markers *MarkerChunk
	Instrument *InstrumentChunk
	Application []*ApplicationChunk
}