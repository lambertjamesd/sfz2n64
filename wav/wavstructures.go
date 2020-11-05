package wav

const (
	FORMAT_PCM = 1
)

const RIFF_HEADER = 0x52494646
const FORMAT_HEADER = 0x666d7420
const DATA_HEADER = 0x64617461
const WAVE_FORMAT = 0x57415645

type WaveHeader struct {
	Format        uint16
	NChannels     uint16
	SampleRate    uint32
	ByteRate      uint32
	BlockAlign    uint16
	BitsPerSample uint16
}

type Wave struct {
	Header WaveHeader
	Data   []byte
}
