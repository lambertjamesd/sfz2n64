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

	if int(8*order*npredictors)+2 != len(chunks) {
		return nil, errors.New(fmt.Sprintf(
			"Wrong number of values for code book expected %d got %d",
			int(8*order*npredictors),
			len(chunks)+2,
		))
	}

	var inputIndex = 2

	for predictor := 0; predictor < int(npredictors); predictor = predictor + 1 {
		codebook.Predictors[predictor] = createPredictor(int(order))

		for i := 0; i < 8; i = i + 1 {
			for orderIndex := 0; orderIndex < int(order); orderIndex = orderIndex + 1 {
				val, err := strconv.ParseInt(chunks[inputIndex], 10, 32)
				inputIndex = inputIndex + 1

				if err != nil {
					return nil, err
				}

				codebook.Predictors[predictor].Table[i][orderIndex] = int32(val)
			}
		}

		ExpandPredictor(&codebook.Predictors[predictor], int(order))
	}

	return &codebook, nil
}

func readPcm(pcmData []int16, start int, length int) []int16 {
	var dataToRead = length

	if start+length > len(pcmData) {
		dataToRead = len(pcmData) - start
	}

	if dataToRead == 0 {
		return make([]int16, length)
	} else if dataToRead == length {
		return pcmData[start : start+dataToRead]
	} else {
		return append(pcmData[start:start+dataToRead], make([]int16, length-dataToRead)...)
	}
}

func acVect(input []int16, order int, frameSize int, out []float64) {
	for i := 0; i <= order; i = i + 1 {
		out[i] = 0
		for j := 0; j < frameSize; j = j + 1 {
			out[i] -= float64(input[frameSize+j-i]) * float64(input[frameSize+j])
		}
	}
}

func acMat(input []int16, order int, frameSize int, out [][]float64) {
	for i := 1; i <= order; i = i + 1 {
		for j := 1; j <= order; j = j + 1 {
			out[i][j] = 0
			for k := 0; k < frameSize; k = k + 1 {
				out[i][j] += float64(input[frameSize+k-i]) * float64(input[frameSize+k-j])
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

func rfroma(arg0 []float64, n int, arg2 []float64) {
	var i, j int
	var mat [][]float64
	var div float64

	mat = make([][]float64, n+1)
	mat[n] = make([]float64, n+1)
	mat[n][0] = 1.0
	for i = 1; i <= n; i++ {
		mat[n][i] = -arg0[i]
	}

	for i = n; i >= 1; i-- {
		mat[i-1] = make([]float64, i)
		div = 1.0 - mat[i][i]*mat[i][i]
		for j = 1; j <= i-1; j++ {
			mat[i-1][j] = (mat[i][i-j]*mat[i][i] + mat[i][j]) / div
		}
	}

	arg2[0] = 1
	for i = 1; i <= n; i++ {
		arg2[i] = 0
		for j = 1; j <= i; j++ {
			arg2[i] += mat[i][j] * arg2[i-j]
		}
	}
}

func durbin(arg0 []float64, n int, arg2 []float64, arg3 []float64, outSomething *float64) int {
	var i, j int
	var sum, div float64
	var ret int

	arg3[0] = 1.0
	div = arg0[0]
	ret = 0

	for i = 1; i <= n; i++ {
		sum = 0.0
		for j = 1; j <= i-1; j++ {
			sum += arg3[j] * arg0[i-j]
		}

		if div > 0 {
			arg3[i] = -(arg0[i] + sum) / div
		} else {
			arg3[i] = 0
		}
		arg2[i] = arg3[i]

		if math.Abs(arg2[i]) > 1 {
			ret++
		}

		for j = 1; j < i; j++ {
			arg3[j] += arg3[i-j] * arg3[i]
		}

		div *= 1.0 - arg3[i]*arg3[i]
	}
	*outSomething = div
	return ret
}

func split(table [][]float64, delta []float64, order int, npredictors int, scale float64) {
	for i := 0; i < npredictors; i++ {
		for j := 0; j <= order; j++ {
			table[i+npredictors][j] = table[i][j] + delta[j]*scale
		}
	}
}

func modelDist(arg0 []float64, arg1 []float64, n int) float64 {
	var ret float64

	var sp3C = make([]float64, n+1)
	var sp38 = make([]float64, n+1)

	rfroma(arg1, n, sp3C)

	for i := 0; i <= n; i++ {
		sp38[i] = 0.0
		for j := 0; j <= n-i; j++ {
			sp38[i] += arg0[j] * arg0[i+j]
		}
	}

	ret = sp38[0] * sp3C[0]
	for i := 1; i <= n; i++ {
		ret += 2 * sp3C[i] * sp38[i]
	}
	return ret
}

func refine(table [][]float64, order int, npredictors int, data [][]float64, dataSize int, refineIters int) {
	var dist float64
	var dummy float64 // spC0
	var bestValue float64
	var bestIndex int

	var rsums = make([][]float64, npredictors)
	for i := 0; i < npredictors; i++ {
		rsums[i] = make([]float64, order+1)
	}

	var counts = make([]float64, npredictors)
	var temp_s7 = make([]float64, order+1)

	for iter := 0; iter < refineIters; iter++ {
		for i := 0; i < npredictors; i++ {
			counts[i] = 0
			for j := 0; j <= order; j++ {
				rsums[i][j] = 0.0
			}
		}

		for i := 0; i < dataSize; i++ {
			bestValue = 1e30
			bestIndex = 0

			for j := 0; j < npredictors; j++ {
				dist = modelDist(table[j], data[i], order)
				if dist < bestValue {
					bestValue = dist
					bestIndex = j
				}
			}

			counts[bestIndex]++
			rfroma(data[i], order, temp_s7)
			for j := 0; j <= order; j++ {
				rsums[bestIndex][j] += temp_s7[j]
			}
		}

		for i := 0; i < npredictors; i++ {
			if counts[i] > 0 {
				for j := 0; j <= order; j++ {
					rsums[i][j] /= counts[i]
				}
			}
		}

		for i := 0; i < npredictors; i++ {
			durbin(rsums[i], order, temp_s7, table[i], &dummy)

			for j := 1; j <= order; j++ {
				if temp_s7[j] >= 1.0 {
					temp_s7[j] = 0.9999999999
				}
				if temp_s7[j] <= -1.0 {
					temp_s7[j] = -0.9999999999
				}
			}

			afromk(temp_s7, table[i], order)
		}
	}
}

func buildPredictor(row []float64, order int) (Predictor, int) {
	var result Predictor

	for i := 0; i < PREDICTOR_SIZE; i++ {
		result.Table[i] = make([]int32, order+PREDICTOR_SIZE)
	}

	var table = make([][]float64, 8)

	for i := 0; i < 8; i++ {
		table[i] = make([]float64, order)
	}

	for i := 0; i < order; i++ {
		for j := 0; j < i; j++ {
			table[i][j] = 0.0
		}

		for j := i; j < order; j++ {
			table[i][j] = -row[order-j+i]
		}
	}

	for i := order; i < 8; i++ {
		for j := 0; j < order; j++ {
			table[i][j] = 0.0
		}
	}

	for i := 1; i < 8; i++ {
		for j := 1; j <= order; j++ {
			if i-j >= 0 {
				for k := 0; k < order; k++ {
					table[i][k] -= row[j] * table[i-j][k]
				}
			}
		}
	}

	var overflows = 0
	for i := 0; i < order; i++ {
		for j := 0; j < 8; j++ {
			var fval = table[j][i] * 2048
			var ival int32
			if fval < 0.0 {
				ival = int32(fval - 0.5)
				if ival < -0x8000 {
					overflows++
				}
			} else {
				ival = int32(fval + 0.5)
				if ival >= 0x8000 {
					overflows++
				}
			}

			result.Table[j][i] = ival
		}
	}

	return result, overflows
}

type CompressionSettings struct {
	Order       int
	FrameSize   int
	Threshold   float64
	Bits        int
	RefineIters int
}

func DefaultCompressionSettings() CompressionSettings {
	return CompressionSettings{
		2,
		16,
		10,
		2,
		2,
	}
}

func CalculateCodebook(pcmData []int16, settings *CompressionSettings) (*Codebook, error) {
	var curr = 0
	var vec = make([]float64, settings.Order+1)

	var mat = make([][]float64, settings.Order+1)

	var spF4 = make([]float64, settings.Order+1)
	var data = make([][]float64, len(pcmData))

	for i := 0; i <= settings.Order; i = i + 1 {
		mat[i] = make([]float64, settings.Order+1)
	}

	var dataSize = 0
	var runningFrame = make([]int16, settings.FrameSize*2)

	for ; curr < len(pcmData); curr = curr + settings.FrameSize {
		var frame = readPcm(pcmData, curr, settings.FrameSize)

		for i := 0; i < settings.FrameSize; i++ {
			runningFrame[i+settings.FrameSize] = frame[i]
		}

		acVect(runningFrame, settings.Order, settings.FrameSize, vec)

		if math.Abs(vec[0]) > settings.Threshold {
			acMat(runningFrame, settings.Order, settings.FrameSize, mat)

			var permDet int
			var perm = make([]int, settings.Order+1)
			if !luDecomp(mat, settings.Order, perm, &permDet) {
				luDecompBackSub(mat, settings.Order, perm, vec)
				vec[0] = 1
				if kfroma(vec, spF4, settings.Order) == 0 {
					data[dataSize] = make([]float64, settings.Order+1)
					data[dataSize][0] = 1.0

					for i := 1; i <= settings.Order; i = i + 1 {
						if spF4[i] >= 1 {
							spF4[i] = 0.9999999999
						}
						if spF4[i] <= -1 {
							spF4[i] = -0.9999999999
						}
					}

					afromk(spF4, data[dataSize], settings.Order)
					dataSize = dataSize + 1
				}
			}
		}

		for i := 0; i < settings.FrameSize; i++ {
			runningFrame[i] = runningFrame[i+settings.FrameSize]
		}
	}

	vec[0] = 1.0
	for j := 1; j <= settings.Order; j++ {
		vec[j] = 0.0
	}

	var temp_s1 [][]float64 = make([][]float64, 1<<settings.Bits)

	for i := 0; i < (1 << settings.Bits); i++ {
		temp_s1[i] = make([]float64, settings.Order+1)
	}

	for i := 0; i < dataSize; i++ {
		rfroma(data[i], settings.Order, temp_s1[0])
		for j := 1; j <= settings.Order; j++ {
			vec[j] += temp_s1[0][j]
		}
	}

	for j := 1; j <= settings.Order; j++ {
		vec[j] = vec[j] / float64(dataSize)
	}

	var dummy float64
	durbin(vec, settings.Order, spF4, temp_s1[0], &dummy)

	for j := 1; j <= settings.Order; j++ {
		if spF4[j] >= 1.0 {
			spF4[j] = 0.9999999999
		}

		if spF4[j] <= -1.0 {
			spF4[j] = -0.9999999999
		}
	}

	afromk(spF4, temp_s1[0], settings.Order)
	var curBits = 0
	var splitDelta []float64 = make([]float64, settings.Order+1)
	for curBits < settings.Bits {
		for i := 0; i <= settings.Order; i++ {
			splitDelta[i] = 0.0
		}
		splitDelta[settings.Order-1] = -1.0
		split(temp_s1, splitDelta, settings.Order, 1<<curBits, 0.01)
		curBits++
		refine(temp_s1, settings.Order, 1<<curBits, data, dataSize, settings.RefineIters)
	}

	var npredictors = 1 << curBits
	var numOverflows = 0

	var result Codebook

	result.Order = settings.Order

	for i := 0; i < npredictors; i++ {
		predector, overflows := buildPredictor(temp_s1[i], settings.Order)

		numOverflows = numOverflows + overflows
		result.Predictors = append(result.Predictors, predector)

		ExpandPredictor(&result.Predictors[i], settings.Order)
	}

	if numOverflows > 0 {
		return &result, errors.New("There was overflow - check the table")
	} else {
		return &result, nil
	}
}
