package convert

import (
	"fmt"
	"os"
)

type writeIntoIns func(state *insConversionState, source interface{}, output *os.File) (string, error)

type insConversionState struct {
	cwd             string
	nameHint        string
	sampleRate      uint32
	usedNames       map[string]bool
	alreadyWritten  map[interface{}]string
	tblData         []byte
	instrumentNames []string
}

func (state *insConversionState) getInstrumentName(index int) string {
	if index < len(state.instrumentNames) {
		return state.instrumentNames[index]
	} else {
		return MIDINames[index]
	}
}

func fixName(name string) string {
	var result []rune = nil

	var first = true

	for _, char := range name {
		if first {
			if char >= '0' && char <= '9' {
				result = append(result, '_')
			}

			first = false
		}

		if char >= '0' && char <= '9' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' {
			result = append(result, char)
		} else if char == ' ' {
			result = append(result, '_')
		}
	}

	return string(result)
}

func (state *insConversionState) getUniqueName(ext string) string {
	var index = 1
	var fixedName = fixName(state.nameHint) + ext

	var searching = true

	for searching {
		_, has := state.usedNames[fixedName]

		if has {
			index = index + 1
			fixedName = fmt.Sprintf("%s%d", fixName(state.nameHint), index) + ext
		} else {
			searching = false
			state.usedNames[fixedName] = true
		}
	}

	return fixedName
}

func (state *insConversionState) writeSection(source interface{}, output *os.File, nameHint string, writer writeIntoIns) (string, error) {
	written, ok := state.alreadyWritten[source]

	if !ok {
		var prevHint = state.nameHint
		state.nameHint = nameHint
		name, err := writer(state, source, output)
		state.nameHint = prevHint

		if err != nil {
			return "", err
		}

		state.alreadyWritten[source] = name
		written = name
	}

	return written, nil
}
