package adpcm

func innerProduct(length int, v1 []int32, v2 [16]int32) int32 {
	var out int32 = 0
	for i := 0; i < length; i = i + 1 {
		out += v1[i] * v2[i]
	}

	// Compute "out / 2^11", rounded down.
	var dout = out / (1 << 11)
	var fiout = dout * (1 << 11)

	if out-fiout < 0 {
		dout = dout - 1
	}

	return dout
}

const MAX_LEVEL = 7

func decodeFrame(frame *Frame, codebook *Codebook, order int, state []int32) {
	var ix [16]int32

	var scale = 1 << (frame.Header >> 4)
	var optimalp = frame.Header & 0xf

	for i := 0; i < 16; i = i + 2 {
		var c = frame.Data[i/2]
		ix[i] = int32(c) >> 4
		ix[i+1] = int32(c) & 0xf
	}

	for i := 0; i < 16; i = i + 1 {
		if ix[i] <= MAX_LEVEL {
			ix[i] = ix[i] * int32(scale)
		} else {
			ix[i] = ix[i] - 16
			ix[i] = ix[i] * int32(scale)
		}
	}

	for j := 0; j < 2; j++ {
		var inputVector [16]int32

		for i := 0; i < 8; i = i + 1 {
			inputVector[i+order] = ix[j*8+i]
		}

		if j == 0 {
			for i := 0; i < order; i = i + 1 {
				inputVector[i] = state[16-order+i]
			}
		} else {
			for i := 0; i < order; i = i + 1 {
				inputVector[i] = state[8-order+i]
			}
		}

		for i := 0; i < 8; i = i + 1 {
			state[j*8+i] = innerProduct(
				order+8,
				codebook.Predictors[optimalp].Table[i],
				inputVector,
			)
		}
	}
}

func fabs(x float32) float32 {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func iabs(x int32) int32 {
	if x < 0 {
		return -x
	} else {
		return x
	}
}

func qsample(x float32, scale int32) int16 {
	if x > 0 {
		return int16((x / float32(scale)) + 0.4999999)
	} else {
		return int16((x / float32(scale)) - 0.4999999)
	}
}

func clamp(fs int32, e [16]float32, ie [16]int32, bits int32) {
	var lowerLevel = 1 << (bits - 1)
	var llevel = -float32(lowerLevel)
	var ulevel = -llevel - 1
	for i := int32(0); i < fs; i = i + 1 {
		if e[i] > ulevel {
			e[i] = ulevel
		}
		if e[i] < llevel {
			e[i] = llevel
		}

		if e[i] > 0 {
			ie[i] = int32(float64(e[i] + 0.5))
		} else {
			ie[i] = int32(float64(e[i] - 0.5))
		}
	}
}

func clip(ix int32, llevel int32, ulevel int32) int32 {
	if ix < llevel || ix > ulevel {
		if ix < llevel {
			return llevel
		}
		if ix > ulevel {
			return ulevel
		}
	}
	return ix
}

func encodeFrame(input []int16, state []int32, codebook *Codebook, order int) *Frame {
	var ix [16]int16
	var inBuffer [16]int16

	var prediction [16]int32
	var inVector [16]int32
	var saveState [16]int32
	var ie [16]int32

	var optimalp int32
	var scale int32
	var llevel int32
	var ulevel int32
	var nIter int32
	var max int32
	var cV int16
	var maxClip int32

	var result Frame
	var e [16]float32
	var se float32
	var min float32

	// We are only given 'nsam' samples; pad with zeroes to 16.
	for i := 0; i < 16; i = i + 1 {
		if i < len(input) {
			inBuffer[i] = input[i]
		} else {
			inBuffer[i] = 0
		}
	}

	llevel = -8
	ulevel = -llevel - 1

	// Determine the best-fitting predictor.
	min = 1e30
	optimalp = 0
	for k := 0; k < len(codebook.Predictors); k = k + 1 {
		// Copy over the last 'order' samples from the previous output.
		for i := 0; i < order; i = i + 1 {
			inVector[i] = state[16-order+i]
		}

		// For 8 samples...
		for i := 0; i < 8; i = i + 1 {
			// Compute a prediction based on 'order' values from the old state,
			// plus previous errors in this chunk, as an inner product with the
			// coefficient table.
			prediction[i] = innerProduct(order+i, codebook.Predictors[k].Table[i], inVector)
			// Record the error in inVector (thus, its first 'order' samples
			// will contain actual values, the rest will be error terms), and
			// in floating point form in e (for no particularly good reason).
			inVector[i+order] = int32(inBuffer[i]) - prediction[i]
			e[i] = float32(inVector[i+order])
		}

		// For the next 8 samples, start with 'order' values from the end of
		// the previous 8-sample chunk of inBuffer. (The code is equivalent to
		// inVector[i] = inBuffer[8 - order + i].)
		for i := 0; i < order; i = i + 1 {
			inVector[i] = prediction[8-order+i] + inVector[8+i]
		}

		// ... and do the same thing as before to get predictions.
		for i := 0; i < 8; i = i + 1 {
			prediction[8+i] = innerProduct(order+i, codebook.Predictors[k].Table[i], inVector)
			inVector[i+order] = int32(inBuffer[8+i]) - prediction[8+i]
			e[8+i] = float32(inVector[i+order])
		}

		// Compute the L2 norm of the errors; the lowest norm decides which
		// predictor to use.
		se = 0
		for j := 0; j < 16; j = j + 1 {
			se += e[j] * e[j]
		}

		if se < min {
			min = se
			optimalp = int32(k)
		}
	}

	// Do exactly the same thing again, for real.
	for i := 0; i < order; i = i + 1 {
		inVector[i] = state[16-order+i]
	}

	for i := 0; i < 8; i = i + 1 {
		prediction[i] = innerProduct(order+i, codebook.Predictors[optimalp].Table[i], inVector)
		inVector[i+order] = int32(inBuffer[i]) - prediction[i]
		e[i] = float32(inVector[i+order])
	}

	for i := 0; i < order; i = i + 1 {
		inVector[i] = prediction[8-order+i] + inVector[8+i]
	}

	for i := 0; i < 8; i = i + 1 {
		prediction[8+i] = innerProduct(order+i, codebook.Predictors[optimalp].Table[i], inVector)
		inVector[i+order] = int32(inBuffer[8+i]) - prediction[8+i]
		e[8+i] = float32(inVector[i+order])
	}

	// Clamp the errors to 16-bit signed ints, and put them in ie.
	clamp(16, e, ie, 16)

	// Find a value with highest absolute value.
	// @bug If this first finds -2^n and later 2^n, it should set 'max' to the
	// latter, which needs a higher value for 'scale'.
	max = 0
	for i := 0; i < 16; i = i + 1 {
		if iabs(ie[i]) > iabs(max) {
			max = ie[i]
		}
	}

	// Compute which power of two we need to scale down by in order to make
	// all values representable as 4-bit signed integers (i.e. be in [-8, 7]).
	// The worst-case 'max' is -2^15, so this will be at most 12.
	for scale := 0; scale <= 12; scale = scale + 1 {
		if max <= ulevel && max >= llevel {
			break
		}
		max /= 2
	}

	for i := 0; i < 16; i = i + 1 {
		saveState[i] = state[i]
	}

	// Try with the computed scale, but if it turns out we don't fit in 4 bits
	// (if some |cV| >= 2), use scale + 1 instead (i.e. downscaling by another
	// factor of 2).
	scale--
	nIter = 0

	var isLooping = true

	for isLooping {
		nIter++
		maxClip = 0
		scale++
		if scale > 12 {
			scale = 12
		}

		// Copy over the last 'order' samples from the previous output.
		for i := 0; i < order; i = i + 1 {
			inVector[i] = saveState[16-order+i]
		}

		// For 8 samples...
		for i := 0; i < 8; i = i + 1 {
			// Compute a prediction based on 'order' values from the old state,
			// plus previous *quantized* errors in this chunk (because that's
			// all the decoder will have available).
			prediction[i] = innerProduct(order+i, codebook.Predictors[optimalp].Table[i], inVector)

			// Compute the error, and divide it by 2^scale, rounding to the
			// nearest integer. This should ideally result in a 4-bit integer.
			se = float32(inBuffer[i]) - float32(prediction[i])
			ix[i] = qsample(se, 1<<scale)

			// Clamp the error to a 4-bit signed integer, and record what delta
			// was needed for that.
			cV = int16(clip(int32(ix[i]), llevel, ulevel)) - int16(ix[i])
			if maxClip < iabs(int32(cV)) {
				maxClip = iabs(int32(cV))
			}
			ix[i] += cV

			// Record the quantized error in inVector for later predictions,
			// and the quantized (decoded) output in state (for use in the next
			// batch of 8 samples).
			inVector[i+order] = int32(ix[i]) * (1 << scale)
			state[i] = prediction[i] + inVector[i+order]
		}

		// Copy over the last 'order' decoded samples from the above chunk.
		for i := 0; i < order; i = i + 1 {
			inVector[i] = state[8-order+i]
		}

		// ... and do the same thing as before.
		for i := 0; i < 8; i = i + 1 {
			prediction[8+i] = innerProduct(order+i, codebook.Predictors[optimalp].Table[i], inVector)
			se = float32(inBuffer[8+i]) - float32(prediction[8+i])
			ix[8+i] = qsample(se, 1<<scale)
			cV = int16(clip(int32(ix[8+i]), llevel, ulevel)) - int16(ix[8+i])
			if maxClip < iabs(int32(cV)) {
				maxClip = iabs(int32(cV))
			}
			ix[8+i] += cV
			inVector[i+order] = int32(ix[8+i]) * (1 << scale)
			state[8+i] = prediction[8+i] + inVector[i+order]
		}

		isLooping = maxClip >= 2 && nIter < 2
	}

	result.Header = uint8(scale<<4) | uint8(optimalp&0xf)

	for i := 0; i < 16; i = i + 2 {
		result.Data[i/2] = uint8(ix[i]<<4) | uint8(ix[i+1]&0xf)
	}

	return &result
}
