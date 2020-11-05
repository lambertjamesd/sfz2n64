package wav

import (
	"encoding/binary"
	"errors"
	"os"
)

type SeekableReader interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
}

func parseHeader(reader SeekableReader, header *WaveHeader) {
	binary.Read(reader, binary.LittleEndian, &header.Format)
	binary.Read(reader, binary.LittleEndian, &header.NChannels)
	binary.Read(reader, binary.LittleEndian, &header.SampleRate)
	binary.Read(reader, binary.LittleEndian, &header.ByteRate)
	binary.Read(reader, binary.LittleEndian, &header.BlockAlign)
	binary.Read(reader, binary.LittleEndian, &header.BitsPerSample)
}

func parseData(reader SeekableReader, len int32) []byte {
	var result = make([]byte, len)
	reader.Read(result)
	return result
}

func Parse(reader SeekableReader) (*Wave, error) {
	var result Wave

	var header int32
	err := binary.Read(reader, binary.BigEndian, &header)

	if err != nil {
		return nil, err
	}

	if header != RIFF_HEADER {
		return nil, errors.New("Invalid wav header")
	}

	var chunkSize int32
	binary.Read(reader, binary.LittleEndian, &chunkSize)

	binary.Read(reader, binary.BigEndian, &header)

	if header != WAVE_FORMAT {
		return nil, errors.New("Invalid wave format")
	}

	var hasHeader = false
	var hasData = false

	for !hasHeader || !hasData {
		err = binary.Read(reader, binary.BigEndian, &header)

		if err != nil {
			return nil, err
		}

		binary.Read(reader, binary.LittleEndian, &chunkSize)

		startPos, _ := reader.Seek(0, os.SEEK_CUR)

		if header == FORMAT_HEADER {
			parseHeader(reader, &result.Header)
			hasHeader = true
		} else if header == DATA_HEADER {
			result.Data = parseData(reader, chunkSize)
			hasData = true
		}

		reader.Seek(startPos+int64(chunkSize), os.SEEK_SET)
	}

	return &result, nil
}
