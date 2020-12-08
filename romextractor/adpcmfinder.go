package romextractor

import (
	"errors"
	"sort"

	"github.com/lambertjamesd/sfz2n64/al64"
)

const FRAME_SIZE = 9

type adpcmFinder struct {
	npredictors    int
	offset         int
	dataLength     int
	frameBlacklist [FRAME_SIZE]int
}

func (finder *adpcmFinder) couldBeHeader(data byte) bool {
	var scale = data >> 4
	var optimalp = data & 0xf

	return scale <= 12 && int(optimalp) < finder.npredictors
}

func (finder *adpcmFinder) findPossibleADPCMLocations(in []byte) map[int]bool {
	var result = make(map[int]bool)

	if len(in) < finder.dataLength {
		return result
	}

	var blackListIndex = 0
	for i := 0; i < finder.dataLength; i++ {
		if !finder.couldBeHeader(in[i]) {
			finder.frameBlacklist[blackListIndex]++
		}

		blackListIndex++
		if blackListIndex == FRAME_SIZE {
			blackListIndex = 0
		}
	}

	var blackListCheck = 0

	for i := finder.dataLength; i <= len(in); i++ {
		var beginningCheck = i - finder.dataLength

		if finder.frameBlacklist[blackListCheck] == 0 && (beginningCheck & ^0x7) == beginningCheck {
			result[beginningCheck-finder.offset] = true
		}

		if i < len(in) {
			if !finder.couldBeHeader(in[i]) {
				finder.frameBlacklist[blackListIndex]++
			}

			if !finder.couldBeHeader(in[beginningCheck]) {
				finder.frameBlacklist[blackListCheck]--
			}

			blackListIndex++
			if blackListIndex == FRAME_SIZE {
				blackListIndex = 0
			}

			blackListCheck++
			if blackListCheck == FRAME_SIZE {
				blackListCheck = 0
			}
		}
	}

	return result
}

func findPossibleLocations(wavetable *al64.ALWavetable, in []byte) map[int]bool {
	var finder = adpcmFinder{
		int(wavetable.AdpcWave.Book.NPredictors),
		int(wavetable.Base),
		int(wavetable.Len),
		[FRAME_SIZE]int{},
	}

	return finder.findPossibleADPCMLocations(in)
}

type WavetableList []*al64.ALWavetable

func (list WavetableList) Len() int {
	return len(list)
}

func (list WavetableList) Less(i, j int) bool {
	return list[j].Len < list[i].Len
}

func (list WavetableList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func FindTblWithTables(wavetables WavetableList, in []byte) (int, error) {
	if len(wavetables) == 0 {
		return 0, errors.New("No wavetables")
	}

	sort.Sort(wavetables)

	var possibleLocations = findPossibleLocations(wavetables[0], in)

	for i := 1; i < len(wavetables); i++ {
		if len(possibleLocations) == 0 {
			break
		} else if len(possibleLocations) == 1 {
			for index, _ := range possibleLocations {
				return index, nil
			}
		}

		var otherSoundLocations = findPossibleLocations(wavetables[i], in)

		var nextPossibleLocations = make(map[int]bool)

		for index, _ := range possibleLocations {
			isPossibleInOther, _ := otherSoundLocations[index]

			if isPossibleInOther {
				nextPossibleLocations[index] = true
			}
		}

		possibleLocations = nextPossibleLocations
	}

	return 0, errors.New("Could not find any possible location")
}

func listWavetablesInInstrument(instrument *al64.ALInstrument, wavetables WavetableList, maxLength int) (WavetableList, int) {
	if instrument == nil {
		return wavetables, maxLength
	}

	for _, sound := range instrument.SoundArray {
		if sound.Wavetable.Type == al64.AL_ADPCM_WAVE {
			wavetables = append(wavetables, sound.Wavetable)
		}

		var tableLen = int(sound.Wavetable.Base + sound.Wavetable.Len)

		if tableLen > maxLength {
			maxLength = tableLen
		}
	}

	return wavetables, maxLength
}

func FindTbl(bankFile *al64.ALBankFile, in []byte) (int, int, error) {
	var wavetables WavetableList = nil
	var instLen int = 0

	for _, bank := range bankFile.BankArray {
		wavetables, instLen = listWavetablesInInstrument(bank.Percussion, wavetables, instLen)

		for _, instrument := range bank.InstArray {
			wavetables, instLen = listWavetablesInInstrument(instrument, wavetables, instLen)
		}
	}

	start, err := FindTblWithTables(wavetables, in)
	return start, instLen, err
}
