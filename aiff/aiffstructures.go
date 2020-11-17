package aiff

import "math"

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

func (markers *MarkerChunk) FindMarker(id uint16) *Marker {
	for _, marker := range markers.Markers {
		if marker.ID == id {
			return &marker
		}
	}

	return nil
}

func ExtendedFromF64(val float64) ExtendedFloat {
	var asInt = math.Float64bits(val)

	var sign = asInt & 0x8000000000000000
	var exponent = (asInt ^ sign) >> 52
	var mantissa = asInt & 0xFFFFFFFFFFFFF

	exponent = exponent + 0x3FFF - 1023

	mantissa = 0x8000000000000000 | (mantissa << (63 - 52))

	return ExtendedFloat{
		sign != 0,
		uint16(exponent),
		mantissa,
	}
}

func F64FromExtended(val ExtendedFloat) float64 {
	if val.Exponent == 0 && val.Mantissa == 0 {
		return 0
	}

	var sign float64 = 1

	if val.Sign {
		sign = -1
	}

	var mant = float64(val.Mantissa) / math.Pow(2, 63)

	return sign * mant * math.Pow(2, float64(val.Exponent)-0x3FFF)
}
