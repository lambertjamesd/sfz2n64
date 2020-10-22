package aiff

import (
	"encoding/binary"
	"errors"
	"os"
)

type SeekableReader interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
}

func readExtended(reader SeekableReader) (ExtendedFloat, error) {
	var exponent uint16
	err := binary.Read(reader, binary.BigEndian, &exponent)

	if err != nil {
		return ExtendedFloat{}, err
	}

	var mantissa uint64
	err = binary.Read(reader, binary.BigEndian, &mantissa)

	if err != nil {
		return ExtendedFloat{}, err
	}

	return ExtendedFloat{
		(exponent & 0x8000) != 0,
		exponent & 0x7FFF,
		mantissa,
	}, nil
}

func readPString(reader SeekableReader) (string, error) {
	var len uint8
	err := binary.Read(reader, binary.BigEndian, &len)

	if err != nil {
		return "", err
	}

	var buffer = make([]byte, len)
	_, err = reader.Read(buffer)

	if err != nil {
		return "", err
	}

	if len%2 == 0 {
		// read padding byte
		binary.Read(reader, binary.BigEndian, &len)
	}

	return string(buffer), nil
}

func parseCommonChunk(reader SeekableReader, compressed bool) (*CommonChunk, error) {
	var result CommonChunk

	err := binary.Read(reader, binary.BigEndian, &result.NumChannels)

	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &result.NumSampleFrames)

	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &result.SampleSize)

	if err != nil {
		return nil, err
	}

	sampleRate, err := readExtended(reader)

	if err != nil {
		return nil, err
	}

	result.SampleRate = sampleRate

	if compressed {
		err = binary.Read(reader, binary.BigEndian, &result.CompressionType)

		if err != nil {
			return nil, err
		}

		compressionName, err := readPString(reader)

		if err != nil {
			return nil, err
		}

		result.CompressionName = compressionName
	}

	return &result, nil
}

func parseSoundDataChunk(reader SeekableReader, chunkSize uint32) (*SoundDataChunk, error) {
	var result SoundDataChunk

	var offset uint32

	err := binary.Read(reader, binary.BigEndian, &offset)

	if err != nil {
		return nil, err
	}

	result.Offset = offset

	var blockSize uint32

	err = binary.Read(reader, binary.BigEndian, &blockSize)

	if err != nil {
		return nil, err
	}

	result.BlockSize = blockSize

	result.WaveformData = make([]byte, chunkSize-8)

	_, err = reader.Read(result.WaveformData)

	if err != nil {
		return nil, err
	}

	return &result, nil
}

func parseInstrumentChunk(reader SeekableReader) (*InstrumentChunk, error) {
	var result InstrumentChunk

	binary.Read(reader, binary.BigEndian, &result.BaseNote)
	binary.Read(reader, binary.BigEndian, &result.Detune)
	binary.Read(reader, binary.BigEndian, &result.LowNote)
	binary.Read(reader, binary.BigEndian, &result.HighNote)
	binary.Read(reader, binary.BigEndian, &result.LowVelocity)
	binary.Read(reader, binary.BigEndian, &result.HighVelocity)
	binary.Read(reader, binary.BigEndian, &result.Gain)

	binary.Read(reader, binary.BigEndian, &result.SustainLoop.PlayMode)
	binary.Read(reader, binary.BigEndian, &result.SustainLoop.BeginLoop)
	binary.Read(reader, binary.BigEndian, &result.SustainLoop.EndLoop)

	binary.Read(reader, binary.BigEndian, &result.ReleaseLoop.PlayMode)
	binary.Read(reader, binary.BigEndian, &result.ReleaseLoop.BeginLoop)
	err := binary.Read(reader, binary.BigEndian, &result.ReleaseLoop.EndLoop)

	return &result, err
}

func parseApplicationChunk(reader SeekableReader, chunkSize uint32) (*ApplicationChunk, error) {
	var result ApplicationChunk

	binary.Read(reader, binary.BigEndian, &result.Signature)

	result.Data = make([]byte, chunkSize-4)

	_, err := reader.Read(result.Data)

	return &result, err
}

func Parse(reader SeekableReader) (*Aiff, error) {
	var result Aiff

	var id uint32

	err := binary.Read(reader, binary.BigEndian, &id)

	if err != nil {
		return nil, err
	}

	if id != FORM_HEADER {
		return nil, errors.New("File didn't have FORM header")
	}

	var chunkSize uint32
	binary.Read(reader, binary.BigEndian, &chunkSize)

	binary.Read(reader, binary.BigEndian, &id)

	if id == AIFC {
		result.Compressed = true
	} else if id != AIFF {
		return nil, errors.New("File didn't have AIFF or AIFC type")
	}

	var fileDone = false

	for !fileDone {
		err = binary.Read(reader, binary.BigEndian, &id)

		if err != nil {
			fileDone = true
		} else {
			binary.Read(reader, binary.BigEndian, &chunkSize)

			currPos, err := reader.Seek(0, os.SEEK_CUR)

			if err != nil {
				return nil, err
			}

			switch id {
			case COMM:
				result.Common, err = parseCommonChunk(reader, result.Compressed)
			case SSND:
				result.SoundData, err = parseSoundDataChunk(reader, chunkSize)
			case INST:
				result.Instrument, err = parseInstrumentChunk(reader)
			case APPL:
				appl, err := parseApplicationChunk(reader, chunkSize)

				if err != nil {
					return nil, err
				}

				result.Application = append(result.Application, appl)
			}

			if err != nil {
				return nil, err
			}

			reader.Seek(int64(chunkSize)+int64(currPos), os.SEEK_SET)
		}
	}

	return &result, nil
}
