package wav

import (
	"bytes"
	"encoding/binary"
	"io"
)

func generateHeader(header *WaveHeader) []byte {
	var result bytes.Buffer

	binary.Write(&result, binary.LittleEndian, &header.Format)
	binary.Write(&result, binary.LittleEndian, &header.NChannels)
	binary.Write(&result, binary.LittleEndian, &header.SampleRate)
	binary.Write(&result, binary.LittleEndian, &header.ByteRate)
	binary.Write(&result, binary.LittleEndian, &header.BlockAlign)
	binary.Write(&result, binary.LittleEndian, &header.BitsPerSample)

	return result.Bytes()
}

func (wave *Wave) Serialize(out io.Writer) error {
	var header = generateHeader(&wave.Header)

	var headStore uint32
	var chunkSize uint32

	headStore = RIFF_HEADER
	err := binary.Write(out, binary.BigEndian, &headStore)

	if err != nil {
		return err
	}

	chunkSize = uint32(len(header) + len(wave.Data) + 20)
	binary.Write(out, binary.LittleEndian, &chunkSize)

	headStore = WAVE_FORMAT
	binary.Write(out, binary.BigEndian, &headStore)

	headStore = FORMAT_HEADER
	binary.Write(out, binary.BigEndian, &headStore)
	chunkSize = uint32(len(header))
	binary.Write(out, binary.LittleEndian, &chunkSize)
	out.Write(header)

	headStore = DATA_HEADER
	binary.Write(out, binary.BigEndian, &headStore)
	chunkSize = uint32(len(wave.Data))
	binary.Write(out, binary.LittleEndian, &chunkSize)
	_, err = out.Write(wave.Data)
	return err
}
