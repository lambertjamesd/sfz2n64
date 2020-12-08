package romextractor

const PENDING_BUFFER_SIZE = 4

type ByteSwapper func(p []byte)

func NativeByteSwapper(p []byte) {

}

func ByteSwappedByteSwapper(p []byte) {
	for offset := 0; offset < len(p); offset += 4 {
		p[offset+0], p[offset+1], p[offset+2], p[offset+3] = p[offset+1], p[offset+0], p[offset+3], p[offset+2]
	}
}

func LittleEndianSwapper(p []byte) {
	for offset := 0; offset < len(p); offset += 4 {
		p[offset+0], p[offset+1], p[offset+2], p[offset+3] = p[offset+3], p[offset+2], p[offset+1], p[offset+0]
	}
}

func DetermineByteSwapper(header []byte) ByteSwapper {
	if len(header) >= 4 {
		if header[0] == 0x80 && header[1] == 0x37 && header[2] == 0x12 && header[3] == 0x40 {
			return NativeByteSwapper
		} else if header[1] == 0x80 && header[0] == 0x37 && header[3] == 0x12 && header[2] == 0x40 {
			return ByteSwappedByteSwapper
		} else if header[3] == 0x80 && header[2] == 0x37 && header[1] == 0x12 && header[0] == 0x40 {
			return LittleEndianSwapper
		}
	}

	return LittleEndianSwapper
}

func CorrectByteswap(data []byte) {
	var swapper = DetermineByteSwapper(data)

	if swapper != nil {
		swapper(data)
	}
}
