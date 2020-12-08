package al64

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
)

type alParseState struct {
	parsed map[int32]interface{}
}

type alTypeParser func(*alParseState, SeekableReader, int32) (interface{}, error)

type SeekableReader interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (ret int64, err error)
}

// arbitrary max ctl file size used to filter out bad addresses
const maxOffset = 500 * 1024

const maxBankCount = 128
const maxInstrumentCount = 2048
const maxSoundCount = 2048

const minSampleRate = 8000
const maxSampleRate = 96000

func readTypeAt(state *alParseState, reader SeekableReader, address int32, parser alTypeParser) (interface{}, error) {
	if address > maxOffset {
		return nil, errors.New(fmt.Sprintf("Could not read address of %d", address))
	}

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

	var bookCount = result.Order * result.NPredictors * 8

	result.Book = make([]int16, bookCount)

	for i := 0; int32(i) < bookCount; i = i + 1 {
		binary.Read(reader, binary.BigEndian, &result.Book[i])
	}

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
		return nil, errors.New("Expected *ALKeyMap")
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

	if result.Volume >= 128 {
		return nil, errors.New("Invalid value for volume")
	}

	binary.Read(reader, binary.BigEndian, &result.Pan)

	if result.Pan >= 128 {
		return nil, errors.New("Invalid value for pan")
	}

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

	if soundCount < 0 || soundCount > maxSoundCount {
		return nil, errors.New("Invalid sound count")
	}

	result.SoundArray = make([]*ALSound, soundCount)

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

		result.SoundArray[i] = sound
	}

	return &result, nil
}

func readBank(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var instrumentCount int16
	binary.Read(reader, binary.BigEndian, &instrumentCount)

	if instrumentCount < 0 || instrumentCount > maxInstrumentCount {
		return nil, errors.New("Invalid number of instruments")
	}

	var padding uint16
	binary.Read(reader, binary.BigEndian, &padding)

	if padding != 0 {
		return nil, errors.New("Invalid padding")
	}

	var result ALBank
	binary.Read(reader, binary.BigEndian, &result.SampleRate)

	if result.SampleRate < minSampleRate || result.SampleRate > maxSampleRate {
		return nil, errors.New("Invalid sample rate")
	}

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

	result.InstArray = make([]*ALInstrument, instrumentCount)

	for i := 0; int16(i) < instrumentCount; i = i + 1 {
		var instrumentOffset int32
		binary.Read(reader, binary.BigEndian, &instrumentOffset)

		if instrumentOffset != 0 {
			bankInt, err := readTypeAt(state, reader, instrumentOffset, readInstrument)

			if err != nil {
				return nil, err
			}

			inst, didCast := bankInt.(*ALInstrument)

			if !didCast {
				return nil, errors.New("Expected ALInstrument")
			}

			result.InstArray[i] = inst
		} else {
			result.InstArray = append(result.InstArray, nil)
		}

	}

	return &result, nil
}

func readBankFile(state *alParseState, reader SeekableReader, address int32) (interface{}, error) {
	var result ALBankFile

	var revision int16
	err := binary.Read(reader, binary.BigEndian, &revision)

	if err != nil {
		return nil, err
	}

	if revision != BANK_REVISION {
		return nil, errors.New("Bad revision number")
	}

	var bankCount int16
	binary.Read(reader, binary.BigEndian, &bankCount)

	if bankCount < 0 || bankCount > maxBankCount {
		return nil, errors.New(fmt.Sprintf("Invalid value for bankCount %d", bankCount))
	}

	result.BankArray = make([]*ALBank, bankCount)

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

		result.BankArray[i] = bank
	}

	return &result, nil
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

	return result, nil
}
