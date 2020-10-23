package aiff

const FORM_HEADER = 0x464F524D

const AIFC = 0x41494643
const AIFF = 0x41494646

const COMM = 0x434F4D4D
const INST = 0x494E5354
const SSND = 0x53534E44
const APPL = 0x4150504C
const MARK = 0x4D41524B

// Sign * 1.Mantissa * pow(2, Exponent - 0x3FFF)
type ExtendedFloat struct {
	Sign     bool
	Exponent uint16
	Mantissa uint64
}

type CommonChunk struct {
	NumChannels     int16
	NumSampleFrames int32
	SampleSize      int16
	SampleRate      ExtendedFloat
	CompressionType uint32
	CompressionName string
}

type Marker struct {
	ID       uint16
	Position uint32
	Name     string
}

type MarkerChunk struct {
	Markers []Marker
}

type Loop struct {
	PlayMode  int16
	BeginLoop uint16
	EndLoop   uint16
}

type InstrumentChunk struct {
	BaseNote     uint8
	Detune       uint8
	LowNote      uint8
	HighNote     uint8
	LowVelocity  uint8
	HighVelocity uint8
	Gain         int16
	SustainLoop  Loop
	ReleaseLoop  Loop
}

type ApplicationChunk struct {
	Signature uint32
	Data      []byte
}

type SoundDataChunk struct {
	Offset       uint32
	BlockSize    uint32
	WaveformData []byte
}

type chunkData struct {
	header uint32
	data   []byte
}

type Aiff struct {
	Compressed  bool
	Common      *CommonChunk
	SoundData   *SoundDataChunk
	Markers     *MarkerChunk
	Instrument  *InstrumentChunk
	Application []*ApplicationChunk
}
