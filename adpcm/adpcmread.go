package adpcm

import (
	"encoding/binary"
	"io"
)

func ExpandPredictor(codebook *Predictor, order int) {
	for k := 1; k < PREDICTOR_SIZE; k = k + 1 {
		codebook.Table[k][order] = codebook.Table[k-1][order-1]
	}

	codebook.Table[0][order] = 1 << 11

	for k := 1; k < PREDICTOR_SIZE; k = k + 1 {
		var j = 0
		for ; j < k; j = j + 1 {
			codebook.Table[j][k+order] = 0
		}

		for ; j < 8; j = j + 1 {
			codebook.Table[j][k+order] = codebook.Table[j-k][order]
		}
	}
}

func createPredictor(order int) Predictor {
	var result Predictor

	for i := 0; i < PREDICTOR_SIZE; i = i + 1 {
		result.Table[i] = make([]int32, order+PREDICTOR_SIZE)
	}

	return result
}

func readPredictor(predictor *Predictor, order int, reader io.Reader) error {
	for idx := 0; idx < PREDICTOR_SIZE; idx = idx + 1 {
		for orderIdx := 0; orderIdx < order; orderIdx = orderIdx + 1 {
			var entry int16
			err := binary.Read(reader, binary.BigEndian, &entry)

			if err != nil {
				return err
			}

			predictor.Table[idx][orderIdx] = int32(entry)
		}
	}

	return nil
}

func ReadBookFromAIFC(reader io.Reader) (*Codebook, error) {
	var result Codebook

	var order int16
	var npredictors int16

	err := binary.Read(reader, binary.BigEndian, &order)

	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &npredictors)

	if err != nil {
		return nil, err
	}

	result.Predictors = make([]Predictor, npredictors)
	result.Order = int(order)

	for predictor := int16(0); predictor < npredictors; predictor = predictor + 1 {
		result.Predictors[predictor] = createPredictor(int(order))
	}

	for predictor := int16(0); predictor < npredictors; predictor = predictor + 1 {
		err = readPredictor(&result.Predictors[predictor], int(order), reader)

		if err != nil {
			return nil, err
		}

		ExpandPredictor(&result.Predictors[predictor], int(order))
	}

	return &result, nil
}

func ReadFrames(input []byte) []Frame {
	var result []Frame = nil

	for i := 0; i+8 < len(input); i = i + 9 {
		var data [8]uint8

		for dataIdx := 0; dataIdx < 8; dataIdx = dataIdx + 1 {
			data[dataIdx] = uint8(input[i+1+dataIdx])
		}

		result = append(result, Frame{
			input[i],
			data,
		})
	}

	return result
}

func NumberSamples(encodedLength int32) int32 {
	return (encodedLength / 9) * 16
}
