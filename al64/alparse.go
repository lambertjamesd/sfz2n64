package al64

import (
	"encoding/binary"
	"errors"
	"os"
	"sort"
)

type alDeferredBookRead struct {
	book *ALADPCMBook
	addr int32
}

type alParseState struct {
	parsed           map[int32]interface{}
	pendingBookReads []alDeferredBookRead
}

type alTypeParser func(*alParseState, SeekableReader, int32) (interface{}, error)

type SeekableReader interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
}

func readTypeAt(state *alParseState, reader SeekableReader, address int32, parser alTypeParser) (interface{}, error) {
	result, exists := state.parsed[address]

	if exists {
		return result, nil
	} else {
		currPos, err := reader.Seek(0, os.SEEK_CUR)

		if err != nil {
			return nil, err
		}

		_, err = reader.Seek(int64(address), os.SEEK_SET)

		if err != nil {
			return nil, err
		}

		result, err := parser(state, reader, address)

		if err != nil {
			return nil, err
		}

		if state.parsed == nil {
			state.parsed = make(map[int32]interface{})
		}

		state.parsed[address] = result

		_, err = reader.Seek(currPos, os.SEEK_SET)

		if err != nil {
			return nil, err
		}

		return result, nil
	}
}

func readLoop(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALADPCMloop
	binary.Read(reader, binary.BigEndian, &result.Start)
	binary.Read(reader, binary.BigEndian, &result.End)
	binary.Read(reader, binary.BigEndian, &result.Count)

	for i := 0; i < ADPCMFSIZE; i = i + 1 {
		binary.Read(reader, binary.BigEndian, &result.State[i])
	}
	return &result, nil
}

func readBook(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALADPCMBook
	binary.Read(reader, binary.BigEndian, &result.Order)
	binary.Read(reader, binary.BigEndian, &result.NPredictors)

	state.pendingBookReads = append(state.pendingBookReads, alDeferredBookRead{
		&result,
		address,
	})

	return &result, nil
}

func readLoopRaw(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALRawLoop
	binary.Read(reader, binary.BigEndian, &result.Start)
	binary.Read(reader, binary.BigEndian, &result.End)
	binary.Read(reader, binary.BigEndian, &result.Count)
	return &result, nil
}

func readWavetable(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALWavetable

	binary.Read(reader, binary.BigEndian, &result.Base)
	binary.Read(reader, binary.BigEndian, &result.Len)
	binary.Read(reader, binary.BigEndian, &result.Type)
	var flags uint8
	binary.Read(reader, binary.BigEndian, &flags)

	// align
	binary.Read(reader, binary.BigEndian, &flags)
	binary.Read(reader, binary.BigEndian, &flags)

	if result.Type == AL_ADPCM_WAVE {
		var offset int32
		binary.Read(reader, binary.BigEndian, &offset)

		if offset != 0 {
			interfaceCheck, err := readTypeAt(state, reader, offset, readLoop)

			if err != nil {
				return nil, err
			}

			loop, ok := interfaceCheck.(*ALADPCMloop)

			if !ok {
				return nil, errors.New("Expected *ALADPCMloop")
			}

			result.AdpcWave.Loop = loop
		} else {
			result.AdpcWave.Loop = nil
		}

		binary.Read(reader, binary.BigEndian, &offset)

		if offset == 0 {
			return nil, errors.New("Null ADPC Book")
		}

		interfaceCheck, err := readTypeAt(state, reader, offset, readBook)

		if err != nil {
			return nil, err
		}

		book, ok := interfaceCheck.(*ALADPCMBook)

		if !ok {
			return nil, errors.New("Expected *ALADPCMBook")
		}

		result.AdpcWave.Book = book
	} else if result.Type == AL_RAW16_WAVE {
		var offset int32
		binary.Read(reader, binary.BigEndian, &offset)

		if offset != 0 {
			interfaceCheck, err := readTypeAt(state, reader, offset, readLoopRaw)

			if err != nil {
				return nil, err
			}

			loop, ok := interfaceCheck.(*ALRawLoop)

			if !ok {
				return nil, errors.New("Expected *ALRawLoop")
			}

			result.RawWave.Loop = loop
		} else {
			result.RawWave.Loop = nil
		}
	} else {
		return nil, errors.New("Unrecognized ALWavetable type")
	}

	return &result, nil
}

func readKeyMap(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALKeyMap

	binary.Read(reader, binary.BigEndian, &result.VelocityMin)
	binary.Read(reader, binary.BigEndian, &result.VelocityMax)
	binary.Read(reader, binary.BigEndian, &result.KeyMin)
	binary.Read(reader, binary.BigEndian, &result.KeyMax)
	binary.Read(reader, binary.BigEndian, &result.KeyBase)
	binary.Read(reader, binary.BigEndian, &result.Detune)

	return &result, nil
}

func readEnvelope(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALEnvelope

	binary.Read(reader, binary.BigEndian, &result.AttackTime)
	binary.Read(reader, binary.BigEndian, &result.DecayTime)
	binary.Read(reader, binary.BigEndian, &result.ReleaseTime)
	binary.Read(reader, binary.BigEndian, &result.AttackVolume)
	binary.Read(reader, binary.BigEndian, &result.DecayVolume)

	return &result, nil
}

func readSound(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALSound

	var offset int32
	binary.Read(reader, binary.BigEndian, &offset)

	if offset == 0 {
		return nil, errors.New("Null Envelope")
	}

	interfaceCheck, err := readTypeAt(state, reader, offset, readEnvelope)

	if err != nil {
		return nil, err
	}

	envelope, didCast := interfaceCheck.(*ALEnvelope)

	if !didCast {
		return nil, errors.New("Expected *ALEnvelope")
	}

	result.Envelope = envelope

	binary.Read(reader, binary.BigEndian, &offset)

	if offset == 0 {
		return nil, errors.New("Null Key Map")
	}

	interfaceCheck, err = readTypeAt(state, reader, offset, readKeyMap)

	if err != nil {
		return nil, err
	}

	keyMap, didCast := interfaceCheck.(*ALKeyMap)

	if !didCast {
		return nil, errors.New("Expected *ALEnvelope")
	}

	result.KeyMap = keyMap

	binary.Read(reader, binary.BigEndian, &offset)

	if offset == 0 {
		return nil, errors.New("Null Wavetable")
	}

	interfaceCheck, err = readTypeAt(state, reader, offset, readWavetable)

	if err != nil {
		return nil, err
	}

	wavetable, didCast := interfaceCheck.(*ALWavetable)

	if !didCast {
		return nil, errors.New("Expected *ALWavetable")
	}

	result.Wavetable = wavetable

	binary.Read(reader, binary.BigEndian, &result.SamplePan)
	binary.Read(reader, binary.BigEndian, &result.SampleVolume)

	return &result, nil
}

func readInstrument(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALInstrument

	binary.Read(reader, binary.BigEndian, &result.Volume)
	binary.Read(reader, binary.BigEndian, &result.Pan)
	binary.Read(reader, binary.BigEndian, &result.Priority)
	var flags uint8
	binary.Read(reader, binary.BigEndian, &flags)

	binary.Read(reader, binary.BigEndian, &result.TremType)
	binary.Read(reader, binary.BigEndian, &result.TremRate)
	binary.Read(reader, binary.BigEndian, &result.TremDepth)
	binary.Read(reader, binary.BigEndian, &result.TremDelay)

	binary.Read(reader, binary.BigEndian, &result.VibType)
	binary.Read(reader, binary.BigEndian, &result.VibRate)
	binary.Read(reader, binary.BigEndian, &result.VibDepth)
	binary.Read(reader, binary.BigEndian, &result.VibDelay)

	binary.Read(reader, binary.BigEndian, &result.BendRange)
	var soundCount int16
	binary.Read(reader, binary.BigEndian, &soundCount)

	for i := 0; int16(i) < soundCount; i = i + 1 {
		var soundOffset int32
		binary.Read(reader, binary.BigEndian, &soundOffset)

		if soundOffset == 0 {
			return nil, errors.New("Null Sound")
		}

		soundInt, err := readTypeAt(state, reader, soundOffset, readSound)

		if err != nil {
			return nil, err
		}

		sound, didCast := soundInt.(*ALSound)

		if !didCast {
			return nil, errors.New("Expected ALSound")
		}

		result.SoundArray = append(result.SoundArray, sound)
	}

	return &result, nil
}

func readBank(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var instrumentCount int16
	binary.Read(reader, binary.BigEndian, &instrumentCount)
	var padding uint16
	binary.Read(reader, binary.BigEndian, &padding)

	var result ALBank
	binary.Read(reader, binary.BigEndian, &result.SampleRate)

	var perucssion int32
	binary.Read(reader, binary.BigEndian, &perucssion)

	if perucssion != 0 {
		bankInt, err := readTypeAt(state, reader, perucssion, readInstrument)

		if err != nil {
			return nil, err
		}

		inst, didCast := bankInt.(*ALInstrument)

		if !didCast {
			return nil, errors.New("Expected ALInstrument")
		}

		result.Percussion = inst
	} else {
		result.Percussion = nil
	}

	for i := 0; int16(i) < instrumentCount; i = i + 1 {
		var instrumentOffset int32
		binary.Read(reader, binary.BigEndian, &instrumentOffset)

		if instrumentOffset == 0 {
			return nil, errors.New("Null Instrument")
		}

		bankInt, err := readTypeAt(state, reader, instrumentOffset, readInstrument)

		if err != nil {
			return nil, err
		}

		inst, didCast := bankInt.(*ALInstrument)

		if !didCast {
			return nil, errors.New("Expected ALInstrument")
		}

		result.InstArray = append(result.InstArray, inst)
	}

	return &result, nil
}

func readBankFile(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALBankFile

	err := binary.Read(reader, binary.BigEndian, &result.Revision)

	if err != nil {
		return nil, err
	}

	if result.Revision != BANK_REVISION {
		return nil, errors.New("Bad revision number")
	}

	var bankCount int16
	binary.Read(reader, binary.BigEndian, &bankCount)

	for i := 0; int16(i) < bankCount; i = i + 1 {
		var addr int32
		binary.Read(reader, binary.BigEndian, &addr)

		if addr == 0 {
			return nil, errors.New("Null ALBank")
		}

		bankInt, err := readTypeAt(state, reader, addr, readBank)

		if err != nil {
			return nil, err
		}

		bank, didCast := bankInt.(*ALBank)

		if !didCast {
			return nil, errors.New("Expected ALBank")
		}

		result.BankArray = append(result.BankArray, bank)
	}

	return &result, nil
}

type chunkLocations []int32

func (arr chunkLocations) Len() int {
	return len(arr)
}

func (arr chunkLocations) Less(i, j int) bool {
	return arr[i] < arr[j]
}

func (arr chunkLocations) Swap(i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}

type alDeferredBookReadArray []alDeferredBookRead

func (arr alDeferredBookReadArray) Len() int {
	return len(arr)
}

func (arr alDeferredBookReadArray) Less(i, j int) bool {
	return arr[i].addr < arr[j].addr
}

func (arr alDeferredBookReadArray) Swap(i, j int) {
	arr[i], arr[j] = arr[j], arr[i]
}

func finishPendingBookReads(state *alParseState, source SeekableReader) error {
	var chunkLocations chunkLocations = nil

	for addr, _ := range state.parsed {
		chunkLocations = append(chunkLocations, addr)
	}

	end, _ := source.Seek(0, os.SEEK_END)

	chunkLocations = append(chunkLocations, int32(end))

	sort.Sort(chunkLocations)

	var pendingBookReads alDeferredBookReadArray = state.pendingBookReads
	sort.Sort(pendingBookReads)

	var currentChunkIndex = 0
	var currentBookRead = 0

	for currentChunkIndex < len(chunkLocations) && currentBookRead < len(pendingBookReads) {
		var chunkAddr = chunkLocations[currentChunkIndex]
		var bookEntry = pendingBookReads[currentBookRead]

		if chunkAddr < bookEntry.addr {
			currentChunkIndex = currentChunkIndex + 1
		} else if chunkAddr > bookEntry.addr {
			currentBookRead = currentBookRead + 1
		} else {
			var bookCount = ((chunkLocations[currentChunkIndex+1] - chunkAddr) - 8) / 2
			source.Seek(int64(chunkAddr+8), os.SEEK_SET)

			for i := 0; int32(i) < bookCount; i = i + 1 {
				var data int16
				binary.Read(source, binary.BigEndian, &data)
				bookEntry.book.Book = append(bookEntry.book.Book, data)
			}

			currentChunkIndex = currentChunkIndex + 1
			currentBookRead = currentBookRead + 1
		}
	}

	return nil
}

func ReadBankFile(source SeekableReader) (*ALBankFile, error) {
	var state alParseState

	interfaceCheck, err := readTypeAt(&state, source, 0, readBankFile)

	if err != nil {
		return nil, err
	}

	result, ok := interfaceCheck.(*ALBankFile)

	if !ok {
		return nil, errors.New("Expected ALBankFile")
	}

	err = finishPendingBookReads(&state, source)

	if err != nil {
		return nil, err
	}

	return result, nil
}
