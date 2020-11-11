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

func luDecomp(a [][]float64, n int, indx []int, d *int) bool {
	var i, imax, j, k int
	var big, dum, sum, temp float64
	var min, max float64
	var vv []float64

	vv = make([]float64, n+1)
	*d = 1
	for i = 1; i <= n; i++ {
		big = 0.0
		for j = 1; j <= n; j++ {
			temp = math.Abs(a[i][j])
			if temp > big {
				big = temp
			}

		}
		if big == 0.0 {
			return true
		}
		vv[i] = 1.0 / big
	}
	for j = 1; j <= n; j++ {
		for i = 1; i < j; i++ {
			sum = a[i][j]
			for k = 1; k < i; k++ {
				sum -= a[i][k] * a[k][j]
			}
			a[i][j] = sum
		}
		big = 0.0
		for i = j; i <= n; i++ {
			sum = a[i][j]
			for k = 1; k < j; k++ {
				sum -= a[i][k] * a[k][j]
			}
			a[i][j] = sum
			dum = vv[i] * math.Abs(sum)
			if dum >= big {
				big = dum
				imax = i
			}
		}
		if j != imax {
			for k = 1; k <= n; k++ {
				dum = a[imax][k]
				a[imax][k] = a[j][k]
				a[j][k] = dum
			}
			*d = -(*d)
			vv[imax] = vv[j]
		}
		indx[j] = imax
		if a[j][j] == 0.0 {
			return true
		}

		if j != n {
			dum = 1.0 / (a[j][j])
			for i = j + 1; i <= n; i++ {
				a[i][j] *= dum
			}
		}
	}

	min = 1e10
	max = 0.0
	for i = 1; i <= n; i++ {
		temp = math.Abs(a[i][i])
		if temp < min {
			min = temp
		}
		if temp > max {
			max = temp
		}
	}

	return min/max < 1e-10
}

func luDecompBackSub(a [][]float64, n int, indx []int, b []float64) {
	var i, ii, ip, j int
	var sum float64

	for i = 1; i <= n; i++ {
		ip = indx[i]
		sum = b[ip]
		b[ip] = b[i]
		if ii != 0 {
			for j = ii; j <= i-1; j++ {
				sum -= a[i][j] * b[j]
			}
		} else if sum != 0 {
			ii = i
		}
		b[i] = sum
	}
	for i = n; i >= 1; i-- {
		sum = b[i]
		for j = i + 1; j <= n; j++ {
			sum -= a[i][j] * b[j]
		}
		b[i] = sum / a[i][i]
	}
}

func afromk(in []float64, out []float64, n int) {
	var i, j int
	out[0] = 1.0
	for i = 1; i <= n; i++ {
		out[i] = in[i]
		for j = 1; j <= i-1; j++ {
			out[j] += out[i-j] * out[i]
		}
	}
}

func kfroma(in []float64, out []float64, n int) int {
	var i, j int
	var div float64
	var temp float64
	var next []float64
	var ret int

	ret = 0
	next = make([]float64, n+1)

	out[n] = in[n]
	for i = n - 1; i >= 1; i-- {
		for j = 0; j <= i; j++ {
			temp = out[i+1]
			div = 1.0 - (temp * temp)
			if div == 0.0 {
				return 1
			}
			next[j] = (in[j] - in[i+1-j]*temp) / div
		}

		for j = 0; j <= i; j++ {
			in[j] = next[j]
		}

		out[i] = next[i]
		if math.Abs(out[i]) > 1.0 {
			ret++
		}
	}

	return ret
}

func CalculateCodebook(pcmData []int16, order int, frameSize int, thresh float64) (*Codebook, error) {
	var curr = 0
	var vec = make([]float64, order+1)

	var mat = make([][]float64, order+1)

	var spF4 = make([]float64, order+1)
	var data = make([][]float64, len(pcmData))

	for i := 0; i <= order; i = i + 1 {
		mat[i] = make([]float64, order+1)
	}

	var dataSize = 0

	for ; curr < len(pcmData); curr = curr + frameSize {
		var frame = readPcm(pcmData, curr, frameSize)
		acVect(frame, order, frameSize, vec)

		if math.Abs(vec[0]) > thresh {
			acMat(frame, order, frameSize, mat)

			var permDet int
			var perm = make([]int, order+1)
			if luDecomp(mat, order, perm, &permDet) {
				luDecompBackSub(mat, order, perm, vec)
				vec[0] = 1
				if kfroma(vec, spF4, order) == 0 {
					data[dataSize] = make([]float64, order+1)
					data[dataSize][0] = 1.0

					for i := 1; i <= order; i = i + 1 {
						if spF4[i] >= 1 {
							spF4[i] = 0.9999999999
						}
						if spF4[i] <= -1 {
							spF4[i] = -0.9999999999
						}
					}

					afromk(spF4, data[dataSize], order)
					dataSize = dataSize + 1
				}
			}
		}
	}

	return nil, nil
}
