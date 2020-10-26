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
		if ix[i] >= 8 {
			ix[i] = ix[i] - 16
		}
		ix[i] = ix[i] * int32(scale)
	}

	for j := 0; j < 2; j++ {
		var inputVector [16]int32
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
			var ind = j*8 + i
			inputVector[order+i] = ix[ind]
			state[ind] = innerProduct(
				order+i,
				codebook.Predictors[optimalp].Table[i],
				inputVector,
			) + ix[ind]
		}
	}
}
