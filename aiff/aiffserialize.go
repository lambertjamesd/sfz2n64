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

	return nil
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

	binary.Write(out, binary.BigEndian, &expSign)
	binary.Write(out, binary.BigEndian, &exFloat.Mantissa)
}

func (commonChunk *CommonChunk) serialize(compressed bool) (*chunkData, error) {
	var result bytes.Buffer

	err := binary.Write(&result, binary.BigEndian, &commonChunk.NumChannels)

	if err != nil {
		return nil, err
	}

	binary.Write(&result, binary.BigEndian, &commonChunk.NumSampleFrames)
	binary.Write(&result, binary.BigEndian, &commonChunk.SampleSize)

	writeExtendedFloat(&result, &commonChunk.SampleRate)

	if compressed {
		binary.Write(&result, binary.BigEndian, &commonChunk.CompressionType)
		err = writePString(&result, commonChunk.CompressionName)

		if err != nil {
			return nil, err
		}
	}

	return &chunkData{COMM, result.Bytes()}, nil
}

func (markerChunk *MarkerChunk) serialize() (*chunkData, error) {
	var result bytes.Buffer

	var count uint16 = uint16(len(markerChunk.Markers))
	binary.Write(&result, binary.BigEndian, &count)

	for _, marker := range markerChunk.Markers {
		binary.Write(&result, binary.BigEndian, &marker.ID)
		binary.Write(&result, binary.BigEndian, &marker.Position)

		writePString(&result, marker.Name)
	}

	return &chunkData{MARK, result.Bytes()}, nil
}

func (instrumentChunk *InstrumentChunk) serialize() (*chunkData, error) {
	var result bytes.Buffer

	binary.Write(&result, binary.BigEndian, &instrumentChunk.BaseNote)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.Detune)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.LowNote)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.HighNote)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.LowVelocity)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.HighVelocity)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.Gain)

	binary.Write(&result, binary.BigEndian, &instrumentChunk.SustainLoop.PlayMode)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.SustainLoop.BeginLoop)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.SustainLoop.EndLoop)

	binary.Write(&result, binary.BigEndian, &instrumentChunk.ReleaseLoop.PlayMode)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.ReleaseLoop.BeginLoop)
	binary.Write(&result, binary.BigEndian, &instrumentChunk.ReleaseLoop.EndLoop)

	return &chunkData{INST, result.Bytes()}, nil
}

func (soundData *SoundDataChunk) serialize() (*chunkData, error) {
	var result bytes.Buffer

	binary.Write(&result, binary.BigEndian, &soundData.Offset)
	binary.Write(&result, binary.BigEndian, &soundData.BlockSize)
	result.Write(soundData.WaveformData)

	return &chunkData{SSND, result.Bytes()}, nil
}

func (applicationData *ApplicationChunk) serialize() (*chunkData, error) {
	var result bytes.Buffer

	binary.Write(&result, binary.BigEndian, &applicationData.Signature)
	result.Write(applicationData.Data)

	return &chunkData{APPL, result.Bytes()}, nil
}

func (aiff *Aiff) serializeChunks() ([]*chunkData, error) {
	var bufferChunks []*chunkData

	commonChunk, err := aiff.Common.serialize(aiff.Compressed)
	if err != nil {
		return nil, err
	}
	bufferChunks = append(bufferChunks, commonChunk)

	if aiff.Markers != nil {
		markerChunk, err := aiff.Markers.serialize()
		if err != nil {
			return nil, err
		}
		bufferChunks = append(bufferChunks, markerChunk)
	}

	if aiff.Instrument != nil {
		instrumentChunk, err := aiff.Instrument.serialize()
		if err != nil {
			return nil, err
		}
		bufferChunks = append(bufferChunks, instrumentChunk)
	}

	for _, appChunk := range aiff.Application {
		applicationChunk, err := appChunk.serialize()
		if err != nil {
			return nil, err
		}
		bufferChunks = append(bufferChunks, applicationChunk)
	}

	soundChunk, err := aiff.SoundData.serialize()
	if err != nil {
		return nil, err
	}
	bufferChunks = append(bufferChunks, soundChunk)

	return bufferChunks, nil
}

func (aiff *Aiff) Serialize(writer io.Writer) error {
	bufferChunks, err := aiff.serializeChunks()

	if err != nil {
		return err
	}

	var totalLength uint32 = 0

	for _, chunk := range bufferChunks {
		totalLength = totalLength + uint32(len(chunk.data)+8)
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
		err := binary.Write(writer, binary.BigEndian, &chunk.header)

		if err != nil {
			return err
		}

		var chunkLen = uint32(len(chunk.data))
		err = binary.Write(writer, binary.BigEndian, &chunkLen)

		if err != nil {
			return err
		}

		_, err = writer.Write(chunk.data)

		if err != nil {
			return err
		}
	}

	return nil
}
