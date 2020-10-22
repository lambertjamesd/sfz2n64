package aiff

import (
	"bytes"
	"encoding/binary"
	"io"
)

func (commonChunk *CommonChunk) serialize() (*bytes.Buffer, error) {
	var result bytes.Buffer

	return &result, nil
}

func (aiff *Aiff) Serialize(writer io.Writer) error {
	var bufferChunks []*bytes.Buffer

	commonChunk, err := aiff.Common.serialize()

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

	for _, chunk := range bufferChunks {

	}

	return nil
}
