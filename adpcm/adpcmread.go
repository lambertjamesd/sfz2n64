package adpcm

import (
	"encoding/binary"
	"io"
)

func expandPredictor(codebook *Predictor) {
	for k := 1; k < PREDICTOR_SIZE; k = k + 1 {
		codebook.Table[k][codebook.Order] = codebook.Table[k-1][codebook.Order-1]
	}

	codebook.Table[0][codebook.Order] = 1 << 11

	for k := 1; k < PREDICTOR_SIZE; k = k + 1 {
		var j = 0
		for ; j < k; j = j + 1 {
			codebook.Table[j][k+codebook.Order] = 0
		}

		for ; j < 8; j = j + 1 {
			codebook.Table[j][k+codebook.Order] = codebook.Table[j-k][codebook.Order]
		}
	}
}

func createPredictor(order int) Predictor {
	var result Predictor

	for i := 0; i < PREDICTOR_SIZE; i = i + 1 {
		result.Table[i] = make([]int32, order+PREDICTOR_SIZE)
	}

	result.Order = order

	return result
}

func readPredictor(predictor *Predictor, reader io.Reader) error {
	for idx := 0; idx < PREDICTOR_SIZE; idx = idx + 1 {
		for orderIdx := 0; orderIdx < predictor.Order; orderIdx = orderIdx + 1 {
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

	for predictor := int16(0); predictor < npredictors; predictor = predictor + 1 {
		result.Predictors[predictor] = createPredictor(int(order))
	}

	for predictor := int16(0); predictor < npredictors; predictor = predictor + 1 {
		err = readPredictor(&result.Predictors[predictor], reader)

		if err != nil {
			return nil, err
		}

		expandPredictor(&result.Predictors[predictor])
	}

	return &result, nil
}
