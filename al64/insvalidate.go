package al64

import "fmt"

func doRangesOverlap(a1, a2, b1, b2 uint8) bool {
	return a1 >= b1 && a1 <= b2 ||
		a2 >= b1 && a2 <= b2 ||
		b1 >= a1 && b1 <= a2 ||
		b2 >= a1 && b2 <= a2
}

func getToken(parseState *parseState, item interface{}, tokens []string) (*Token, string) {
	instrumentMapping, ok := parseState.tokenMapping[item]

	if !ok {
		return nil, ""
	}

	for _, tokenName := range tokens {
		nameLocation, ok := instrumentMapping.namedEntryMapping[tokenName]

		if ok {
			return nameLocation, instrumentMapping.nameToken.value
		}
	}

	return instrumentMapping.nameToken, instrumentMapping.nameToken.value
}

func validateInstrument(instrument *ALInstrument, parseState *parseState, errors []ParseError) []ParseError {
	if instrument == nil {
		return errors
	}

	for index, firstSound := range instrument.SoundArray {
		for secondIndex := 0; secondIndex < index; secondIndex++ {
			var secondSound = instrument.SoundArray[secondIndex]

			if firstSound.KeyMap == nil || secondSound.KeyMap == nil {
				continue
			}

			if doRangesOverlap(
				firstSound.KeyMap.KeyMin,
				firstSound.KeyMap.KeyMax,
				secondSound.KeyMap.KeyMin,
				secondSound.KeyMap.KeyMax,
			) && doRangesOverlap(
				firstSound.KeyMap.VelocityMin,
				firstSound.KeyMap.VelocityMax,
				secondSound.KeyMap.VelocityMin,
				secondSound.KeyMap.VelocityMax,
			) {
				firstSoundLocation, soundName := getToken(parseState, firstSound.KeyMap, []string{"keyMin", "keyMax"})
				secondSoundLocation, secondSoundName := getToken(parseState, secondSound.KeyMap, []string{"keyMin", "keyMax"})
				errors = append(errors, parseState.createError(firstSoundLocation, fmt.Sprintf("keyMin and keyMax overlap in %s", soundName)))
				errors = append(errors, parseState.createError(secondSoundLocation, fmt.Sprintf("with keyMin and keyMax overlap in %s", secondSoundName)))
			}
		}
	}

	return errors
}

func validateIns(bankFile *ALBankFile, parseState *parseState) []ParseError {
	if bankFile == nil {
		return nil
	}

	var result []ParseError = nil

	for _, bank := range bankFile.BankArray {
		result = validateInstrument(bank.Percussion, parseState, result)

		for _, inst := range bank.InstArray {
			result = validateInstrument(inst, parseState, result)
		}
	}

	return result
}
