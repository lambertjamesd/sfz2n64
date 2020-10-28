package convert

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/aiff"
	"github.com/lambertjamesd/sfz2n64/al64"
)

type writeIntoIns func(state *insConversionState, source interface{}, output *os.File) (string, error)

type insConversionState struct {
	cwd            string
	nameHint       string
	sampleRate     uint32
	usedNames      map[string]bool
	alreadyWritten map[interface{}]string
	tblData        []byte
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

func writeWavetable(state *insConversionState, source interface{}, output *os.File) (string, error) {
	wave, ok := source.(*al64.ALWavetable)

	if !ok {
		return "", errors.New("Expected ALWavetable")
	}

	var name string
	var aiffFile aiff.Aiff

	if wave.Type == al64.AL_ADPCM_WAVE {
		name = "." + string(filepath.Separator) + "sounds" + string(filepath.Separator) + state.getUniqueName(".aifc")
		aiffFile.Compressed = true

		aiffFile.Common = &aiff.CommonChunk{
			NumChannels:     1,
			NumSampleFrames: wave.Len * 16 / 9,
			SampleSize:      16,
			SampleRate:      aiff.ExtendedFromF64(float64(state.sampleRate)),
			CompressionType: 0x56415043,
			CompressionName: "VADPCM ~4-1",
		}

		var codesBuffer bytes.Buffer

		var len uint8 = 0xB
		binary.Write(&codesBuffer, binary.BigEndian, &len)
		codesBuffer.WriteString("VADPCMCODES")

		var version uint16 = 1
		binary.Write(&codesBuffer, binary.BigEndian, &version)

		version = uint16(wave.AdpcWave.Book.Order)
		binary.Write(&codesBuffer, binary.BigEndian, &version)
		version = uint16(wave.AdpcWave.Book.NPredictors)
		binary.Write(&codesBuffer, binary.BigEndian, &version)

		for _, val := range wave.AdpcWave.Book.Book {
			binary.Write(&codesBuffer, binary.BigEndian, &val)
		}

		aiffFile.Application = append(aiffFile.Application, &aiff.ApplicationChunk{
			Signature: 0x73746F63,
			Data:      codesBuffer.Bytes(),
		})

		if wave.AdpcWave.Loop != nil {
			var loopBuffer bytes.Buffer

			var len uint8 = 0xB
			binary.Write(&loopBuffer, binary.BigEndian, &len)
			loopBuffer.WriteString("VADPCMLOOPS")

			var version uint16 = 1
			binary.Write(&loopBuffer, binary.BigEndian, &version)
			// num loops
			binary.Write(&loopBuffer, binary.BigEndian, &version)

			binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.Start)
			binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.End)
			binary.Write(&loopBuffer, binary.BigEndian, &wave.AdpcWave.Loop.Count)

			for _, val := range wave.AdpcWave.Loop.State {
				binary.Write(&loopBuffer, binary.BigEndian, &val)
			}

			aiffFile.Application = append(aiffFile.Application, &aiff.ApplicationChunk{
				Signature: 0x73746F63,
				Data:      loopBuffer.Bytes(),
			})
		}
	} else {
		name = "." + string(filepath.Separator) + "sounds" + string(filepath.Separator) + state.getUniqueName(".aiff")
		aiffFile.Compressed = false

		aiffFile.Common = &aiff.CommonChunk{
			NumChannels:     1,
			NumSampleFrames: wave.Len / 2,
			SampleSize:      16,
			SampleRate:      aiff.ExtendedFromF64(44100.0),
			CompressionType: 0,
			CompressionName: "",
		}

		// TODO Loop
	}

	aiffFile.SoundData = &aiff.SoundDataChunk{
		Offset:       0,
		BlockSize:    0,
		WaveformData: state.tblData[wave.Base : wave.Base+wave.Len],
	}

	if _, err := os.Stat(filepath.Join(state.cwd, "sounds")); os.IsNotExist(err) {
		os.Mkdir(filepath.Join(state.cwd, "sounds"), 0776)
	}

	aiffFileOut, err := os.OpenFile(filepath.Join(state.cwd, name), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	if err != nil {
		return name, err
	}

	defer aiffFileOut.Close()

	aiffFile.Serialize(aiffFileOut)

	return name, nil
}

func writeKeyMap(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	keymap, ok := source.(*al64.ALKeyMap)

	if !ok {
		return name, errors.New("Expected ALKeyMap")
	}

	_, err := output.WriteString(fmt.Sprintf("\nkeymap %s\n{\n", name))

	if err != nil {
		return name, err
	}

	output.WriteString(fmt.Sprintf("    velocityMin = %d;\n", keymap.VelocityMin))
	output.WriteString(fmt.Sprintf("    velocityMax = %d;\n", keymap.VelocityMax))
	output.WriteString(fmt.Sprintf("    keyMin = %d;\n", keymap.KeyMin))
	output.WriteString(fmt.Sprintf("    keyMax = %d;\n", keymap.KeyMax))
	output.WriteString(fmt.Sprintf("    keyBase = %d;\n", keymap.KeyBase))
	output.WriteString(fmt.Sprintf("    detune = %d;\n", keymap.Detune))

	_, err = output.WriteString("}\n")

	return name, err
}

func writeEnvelope(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	envelope, ok := source.(*al64.ALEnvelope)

	if !ok {
		return name, errors.New("Expected ALEnvelope")
	}

	_, err := output.WriteString(fmt.Sprintf("\nenvelope %s\n{\n", name))

	if err != nil {
		return name, err
	}

	output.WriteString(fmt.Sprintf("    attackTime = %d;\n", envelope.AttackTime))
	output.WriteString(fmt.Sprintf("    attackVolume = %d;\n", envelope.AttackVolume))
	output.WriteString(fmt.Sprintf("    decayTime = %d;\n", envelope.DecayTime))
	output.WriteString(fmt.Sprintf("    decayVolume = %d;\n", envelope.DecayVolume))
	output.WriteString(fmt.Sprintf("    releaseTime = %d;\n", envelope.ReleaseTime))

	_, err = output.WriteString("}\n")

	return name, err
}

func writeSound(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	sound, ok := source.(*al64.ALSound)

	var soundName string
	var envelopeName string
	var keymapName string

	var err error

	if !ok {
		return name, errors.New("Expected ALSound")
	}

	if sound.Envelope != nil {
		envelopeName, err = state.writeSection(sound.Envelope, output, state.nameHint+"Env", writeEnvelope)

		if err != nil {
			return name, err
		}
	}

	if sound.KeyMap != nil {
		keymapName, err = state.writeSection(sound.KeyMap, output, state.nameHint+"Keymap", writeKeyMap)

		if err != nil {
			return name, err
		}
	}

	if sound.Wavetable != nil {
		soundName, err = state.writeSection(sound.Wavetable, output, state.nameHint+"Snd", writeWavetable)

		if err != nil {
			return name, err
		}
	}

	_, err = output.WriteString(fmt.Sprintf("\nsound %s\n{\n", name))

	if err != nil {
		return name, err
	}

	output.WriteString(fmt.Sprintf("    use(\"%s\");\n", soundName))
	output.WriteString(fmt.Sprintf("    pan = %d;\n", sound.SamplePan))
	output.WriteString(fmt.Sprintf("    volume = %d;\n", sound.SampleVolume))
	output.WriteString(fmt.Sprintf("    keymap = %s;\n", keymapName))
	output.WriteString(fmt.Sprintf("    envelope = %s;\n", envelopeName))

	_, err = output.WriteString("}\n")

	return name, err
}

func writeInstrument(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	inst, ok := source.(*al64.ALInstrument)

	if !ok {
		return name, errors.New("Expected ALInstrument")
	}

	for _, sound := range inst.SoundArray {
		var nextHint = state.nameHint + "Sound"

		if state.nameHint == "Percussion" {
			var key = sound.KeyMap.KeyMin

			if int(key) < len(PercussionNames) {
				nextHint = PercussionNames[key]
			} else {
				nextHint = fmt.Sprintf("Percussion %d", int(key)+1)
			}
		}

		_, err := state.writeSection(sound, output, nextHint, writeSound)

		if err != nil {
			return name, err
		}
	}

	_, err := output.WriteString(fmt.Sprintf("\ninstrument %s\n{\n", name))

	if err != nil {
		return name, err
	}

	output.WriteString(fmt.Sprintf("    volume = %d;\n", inst.Volume))
	output.WriteString(fmt.Sprintf("    pan = %d;\n", inst.Pan))
	output.WriteString(fmt.Sprintf("    priority = %d;\n", inst.Priority))

	if inst.TremType != 0 {
		output.WriteString(fmt.Sprintf("    tremeloType = %d;\n", inst.TremType))
		output.WriteString(fmt.Sprintf("    tremeloRate = %d;\n", inst.TremRate))
		output.WriteString(fmt.Sprintf("    tremeloDepth = %d;\n", inst.TremDepth))
		output.WriteString(fmt.Sprintf("    tremeloDelay = %d;\n", inst.TremDelay))
	}

	if inst.VibType != 0 {
		output.WriteString(fmt.Sprintf("    vibratoType = %d;\n", inst.VibType))
		output.WriteString(fmt.Sprintf("    vibratoRate = %d;\n", inst.VibRate))
		output.WriteString(fmt.Sprintf("    vibratoDepth = %d;\n", inst.VibDepth))
		output.WriteString(fmt.Sprintf("    vibratoDelay = %d;\n", inst.VibDelay))
	}

	if inst.BendRange != 0 {
		output.WriteString(fmt.Sprintf("    bendRange = %d;\n", inst.BendRange))
	}

	for _, sound := range inst.SoundArray {
		name, _ := state.writeSection(sound, output, "", writeSound)
		output.WriteString(fmt.Sprintf("    sound = %s;\n", name))
	}

	_, err = output.WriteString("}\n")

	return name, err
}

func writeALBank(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	alBank, ok := source.(*al64.ALBank)

	if !ok {
		return name, errors.New("Expected ALBank")
	}

	state.sampleRate = alBank.SampleRate

	if alBank.Percussion != nil {
		_, err := state.writeSection(alBank.Percussion, output, "Percussion", writeInstrument)

		if err != nil {
			return name, err
		}
	}

	for index, instrument := range alBank.InstArray {
		if instrument != nil {
			_, err := state.writeSection(instrument, output, MIDINames[index], writeInstrument)

			if err != nil {
				return name, err
			}
		}
	}

	_, err := output.WriteString(fmt.Sprintf("\nbank %s\n{\n    sampleRate = %d;\n", name, alBank.SampleRate))

	if err != nil {
		return name, err
	}

	if alBank.Percussion != nil {
		percussionName, _ := state.writeSection(alBank.Percussion, output, "Percussion", writeInstrument)

		_, err = output.WriteString(fmt.Sprintf("    percussionDefault = %s;\n", percussionName))

		if err != nil {
			return name, err
		}
	}

	for index, instrument := range alBank.InstArray {
		if instrument != nil {
			instrumentName, _ := state.writeSection(instrument, output, MIDINames[index], writeInstrument)

			_, err = output.WriteString(fmt.Sprintf("    program [%d] = %s;\n", index, instrumentName))

			if err != nil {
				return name, err
			}
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
			return "", err
		}
	}

	return "", nil
}

func WriteInsFile(albank *al64.ALBankFile, tblData []byte, filename string) error {
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

	state.usedNames = make(map[string]bool)
	state.alreadyWritten = make(map[interface{}]string)
	state.tblData = tblData

	_, err = state.writeSection(albank, file, nameHint, writeALBankFile)

	return err
}
