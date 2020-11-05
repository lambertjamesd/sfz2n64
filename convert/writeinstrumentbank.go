package convert

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/al64"
)

func writeWavetable(state *insConversionState, source interface{}, output *os.File) (string, error) {
	wave, ok := source.(*al64.ALWavetable)

	if !ok {
		return "", errors.New("Expected ALWavetable")
	}

	var data = state.tblData[wave.Base : wave.Base+wave.Len]

	if wave.Type == al64.AL_ADPCM_WAVE {
		var name = "." + string(filepath.Separator) + "sounds" + string(filepath.Separator) + state.getUniqueName(".aifc")
		var err = writeAifc(filepath.Join(state.cwd, name), wave, data, state.sampleRate)

		err = writeWav(filepath.Join(state.cwd, name[0:len(name)-4]+"wav"), wave, data, state.sampleRate)

		return name, err
	} else {
		var name = "." + string(filepath.Separator) + "sounds" + string(filepath.Separator) + state.getUniqueName(".aiff")
		var err = writeAiff(filepath.Join(state.cwd, name), wave, data, state.sampleRate)
		return name, err
	}
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

func writeInstInstrument(state *insConversionState, source interface{}, output *os.File) (string, error) {
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
		_, err := state.writeSection(alBank.Percussion, output, "Percussion", writeInstInstrument)

		if err != nil {
			return name, err
		}
	}

	for index, instrument := range alBank.InstArray {
		if instrument != nil {
			_, err := state.writeSection(instrument, output, MIDINames[index], writeInstInstrument)

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
		percussionName, _ := state.writeSection(alBank.Percussion, output, "Percussion", writeInstInstrument)

		_, err = output.WriteString(fmt.Sprintf("    percussionDefault = %s;\n", percussionName))

		if err != nil {
			return name, err
		}
	}

	for index, instrument := range alBank.InstArray {
		if instrument != nil {
			instrumentName, _ := state.writeSection(instrument, output, MIDINames[index], writeInstInstrument)

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
