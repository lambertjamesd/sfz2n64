package aiff

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

func writePString(out io.Writer, val string) error {
	var byteLen = len(val)

	if byteLen >= 256 {
		return errors.New("Too long for PString")
	} else {
		var len uint8 = uint8(byteLen)

		err := binary.Write(out, binary.BigEndian, &len)

		if err != nil {
			return err
		}

		asBytes := []byte(val)

		_, err = out.Write(asBytes)

		if err != nil {
			return err
		}

		// check for padding byte
		if len&0x1 == 0x0 {
			len = 0
			err := binary.Write(out, binary.BigEndian, &len)
			if err != nil {
				return err
			}
		}
	}
}

func pStringLen(val string) uint32 {
	var result = 1 + uint32(len(val))

	if result&0x1 == 0x1 {
		return result + 1
	} else {
		return result
	}
}

func writeExtendedFloat(out io.Writer, exFloat *ExtendedFloat) {
	var expSign uint16 = exFloat.Exponent

	if exFloat.Sign {
		expSign = expSign | 0x8000
	}

	binary.Write(out, binary.BigEndian, &exFloat)
	binary.Write(out, binary.BigEndian, &exFloat.Mantissa)
}

func (commonChunk *CommonChunk) serialize(compressed bool) (*bytes.Buffer, error) {
	var result bytes.Buffer

	var header uint32 = COMM
	err := binary.Write(&result, binary.BigEndian, &header)

	if err != nil {
		return nil, err
	}

	var chunkLen uint32

	if compressed {
		chunkLen = 22 + pStringLen(commonChunk.CompressionName)
	} else {
		chunkLen = 18
	}

	err = binary.Write(&result, binary.BigEndian, &chunkLen)

	if err != nil {
		return nil, err
	}

	binary.Write(&result, binary.BigEndian, &commonChunk.NumChannels)
	binary.Write(&result, binary.BigEndian, &commonChunk.NumSampleFrames)
	binary.Write(&result, binary.BigEndian, &commonChunk.SampleSize)

	writeExtendedFloat(&result, &commonChunk.SampleRate)

	if compressed {
		binary.Write(&result, binary.BigEndian, &commonChunk.CompressionType)
		err = writePString(&result, commonChunk.CompressionName)
	}

	return &result, nil
}

func (aiff *Aiff) Serialize(writer io.Writer) error {
	var bufferChunks []*bytes.Buffer

	commonChunk, err := aiff.Common.serialize(aiff.Compressed)

	if err != nil {
		return err
	}

	bufferChunks = append(bufferChunks, commonChunk)

	var totalLength uint32 = 0

	for _, chunk := range bufferChunks {
		totalLength = totalLength + uint32(chunk.Len())
	}

	var header uint32 = FORM_HEADER
	err = binary.Write(writer, binary.BigEndian, &header)

	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, &totalLength)

	if err != nil {
		return err
	}

	if aiff.Compressed {
		header = AIFC
	} else {
		header = AIFF
	}

	err = binary.Write(writer, binary.BigEndian, &header)

	if err != nil {
		return err
	}

	for _, chunk := range bufferChunks {
		_, err := writer.Write(chunk.Bytes())

		if err != nil {
			return err
		}
	}

	return nil
}
