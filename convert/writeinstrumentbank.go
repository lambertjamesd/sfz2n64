package convert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/al64"
)

type writeIntoIns func(state *insConversionState, source interface{}, output *os.File) (string, error)

type insConversionState struct {
	cwd            string
	nameHint       string
	usedNames      map[string]bool
	alreadyWritten map[interface{}]string
}

func fixName(name string) string {
	return name
}

func (state *insConversionState) getUniqueName() string {
	var index = 1
	var fixedName = fixName(state.nameHint)

	var searching = true

	for searching {
		_, has := state.usedNames[fixedName]

		if has {
			index = index + 1
			fixedName = fmt.Sprintf("%s%d", fixName(state.nameHint), index)
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

func writeInstrument(state *insConversionState, source interface{}, output *os.File) (string, error) {
	return "", nil
}

func writeALBank(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName()

	alBank, ok := source.(*al64.ALBank)

	if !ok {
		return name, errors.New("Expected ALBank")
	}

	_, err := output.WriteString(fmt.Sprintf("\nbank %s\n{\n    sampleRate = %d;\n", name, alBank.SampleRate))

	if err != nil {
		return name, err
	}

	if alBank.Percussion != nil {
		percussionName, err := state.writeSection(alBank.Percussion, output, "Percussion", writeInstrument)

		if err != nil {
			return name, err
		}

		_, err = output.WriteString(fmt.Sprintf("    percussionDefault = %s;\n", percussionName))

		if err != nil {
			return name, err
		}
	}

	for index, instrument := range alBank.InstArray {
		instrumentName, err := state.writeSection(instrument, output, MIDINames[index], writeInstrument)

		if err != nil {
			return name, err
		}

		_, err = output.WriteString(fmt.Sprintf("    program[%d] = %s;\n", index, instrumentName))

		if err != nil {
			return name, err
		}
	}

	_, err = output.WriteString("}\n")

	return name, err
}

func writeALBankFile(state *insConversionState, source interface{}, output *os.File) (string, error) {
	alBankFile, ok := source.(*al64.ALBankFile)

	if !ok {
		return "", errors.New("Expected ALBankFile")
	}

	for _, alBank := range alBankFile.BankArray {
		_, err := state.writeSection(alBank, output, state.nameHint, writeALBank)

		if err != nil {
			return "", nil
		}
	}

	return "", nil
}

func WriteInsFile(albank *al64.ALBankFile, filename string) error {
	var state insConversionState

	file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)

	if err != nil {
		return err
	}

	defer file.Close()

	state.cwd = filepath.Dir(filename)
	var nameHint = filepath.Base(filename)
	var ext = filepath.Ext(filename)
	nameHint = nameHint[0 : len(nameHint)-len(ext)]

	state.writeSection(albank, file, nameHint, writeALBankFile)

	return nil
}
