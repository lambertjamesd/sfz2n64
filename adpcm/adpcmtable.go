package adpcm

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"strconv"
	"strings"
)

func (codebook *Codebook) Serialize(out io.Writer) {
	io.WriteString(out, fmt.Sprintf("%d\n%d\n", codebook.Order, len(codebook.Predictors)))

	for _, predictor := range codebook.Predictors {
		for order := 0; order < codebook.Order; order = order + 1 {
			for i := 0; i < PREDICTOR_SIZE; i = i + 1 {
				io.WriteString(out, fmt.Sprintf("%d ", predictor.Table[i][order]))
			}

			io.WriteString(out, "\n")
		}
	}
}

func ParseCodebook(in io.Reader) (*Codebook, error) {
	content, err := ioutil.ReadAll(in)

	if err != nil {
		return nil, err
	}

	var chunks = strings.Fields(string(content))

	if len(chunks) < 2 {
		return nil, errors.New("Missing inforamation in codeboook")
	}

	order, err := strconv.ParseInt(chunks[0], 10, 32)

	if err != nil {
		return nil, err
	}

	npredictors, err := strconv.ParseInt(chunks[1], 10, 32)

	if err != nil {
		return nil, err
	}

	var codebook Codebook

	codebook.Order = int(order)
	codebook.Predictors = make([]Predictor, npredictors)

	if int(8*order*npredictors) != len(chunks)+2 {
		return nil, errors.New(fmt.Sprintf(
			"Wrong number of values for code book expected %d got %d",
			int(8*order*npredictors),
			len(chunks)+2,
		))
	}

	var inputIndex = 2

	for predictor := 0; predictor < int(npredictors); predictor = predictor + 1 {
		for i := 0; i < 8; i = i + 1 {
			codebook.Predictors[predictor].Table[i] = make([]int32, order)

			for orderIndex := 0; orderIndex < int(order); orderIndex = orderIndex + 1 {
				val, err := strconv.ParseInt(chunks[inputIndex], 10, 32)
				inputIndex = inputIndex + 1

				if err != nil {
					return nil, err
				}

				codebook.Predictors[predictor].Table[i][orderIndex] = int32(val)
			}
		}
	}

	return &codebook, nil
}

func readPcm(pcmData []int16, start int, length int) []int16 {
	var dataToRead = length

	if start+length > len(pcmData) {
		dataToRead = start + length - len(pcmData)
	}

	if dataToRead > 0 {
		return make([]int16, length)
	} else if dataToRead == length {
		return pcmData[start : start+dataToRead]
	} else {
		return append(pcmData[start:start+dataToRead], make([]int16, length-dataToRead)...)
	}
}

func acVect(input []int16, n int, m int, out []float64) {
	for i := 0; i < n; i = i + 1 {
		out[i] = 0
		for j := 0; j < m; j = j + 1 {
			out[i] = out[i] - float64(input[j-i]*input[j])
		}
	}
}

func acMat(input []int16, n int, m int, out [][]float64) {
	for i := 1; i <= n; i = i + 1 {
		for j := 1; j <= n; j = j + 1 {
			out[i][j] = 0
			for k := 0; k < m; k = k + 1 {
				out[i][j] = out[i][j] + float64(input[k-i]*input[k-j])
			}
		}
	}
}

func CalculateCodebook(pcmData []int16, order int, frameSize int, thresh float64) (*Codebook, error) {
	var curr = 0
	var vec = make([]float64, order+1)

	var mat = make([][]float64, order+1)

	for i := 0; i <= order; i = i + 1 {
		mat[i] = make([]float64, order+1)
	}

	for ; curr < len(pcmData); curr = curr + frameSize {
		var frame = readPcm(pcmData, curr, frameSize)
		acVect(frame, order, frameSize, vec)

		if math.Abs(vec[0]) > thresh {
			acMat(frame, order, frameSize, mat)
		}
	}

	return nil, nil
}
