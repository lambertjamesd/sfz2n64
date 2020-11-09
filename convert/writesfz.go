package convert

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
)

func writeSfzWavetable(state *insConversionState, source interface{}, output *os.File) (string, error) {
	wave, ok := source.(*al64.ALWavetable)

	if !ok {
		return "", errors.New("Expected ALWavetable")
	}

	var data = state.tblData[wave.Base : wave.Base+wave.Len]

	var name = "." + string(filepath.Separator) + "sounds" + string(filepath.Separator) + state.getUniqueName(".wav")
	var err = audioconvert.WriteWav(filepath.Join(state.cwd, name), wave, data, state.sampleRate)
	return name, err
}

func writeSfzKeyMap(keymap *al64.ALKeyMap, output *os.File) error {
	var finalBase = keymap.KeyBase
	var detune = int8(keymap.Detune)

	output.WriteString(fmt.Sprintf(
		"lokey=%d hikey=%d pitch_keycenter=%d\n",
		keymap.KeyMin,
		keymap.KeyMax,
		finalBase,
	))

	if keymap.VelocityMin != 0 && keymap.VelocityMax != 127 {
		output.WriteString(fmt.Sprintf(
			"lovel=%d hivel=%d\n", keymap.VelocityMin, keymap.VelocityMax,
		))
	}

	if detune != 0 {
		output.WriteString(fmt.Sprintf(
			"tune=%d\n", int(detune),
		))
	}

	return nil
}

func writeSfzEnvelope(envelope *al64.ALEnvelope, output *os.File) error {
	if envelope != nil {
		output.WriteString(fmt.Sprintf("ampeg_attack=%.06f\n", float64(envelope.AttackTime)/1000000))
		output.WriteString(fmt.Sprintf("ampeg_decay=%.06f\n", float64(envelope.DecayTime)/1000000))
		output.WriteString(fmt.Sprintf("ampeg_release=%.06f\n", float64(envelope.ReleaseTime)/1000000))
		if envelope.AttackVolume != 0 {
			output.WriteString(fmt.Sprintf("ampeg_sustain=%.06f\n", 100*float64(envelope.DecayVolume)/float64(envelope.AttackVolume)))
		}
	}
	return nil
}

func writeSfzLoop(start uint32, end uint32, output *os.File) {
	output.WriteString(fmt.Sprintf(`loop_mode=loop_sustain
loop_start=%d
loop_end=%d
`, start, end))
}

func writeSfzInstrument(state *insConversionState, source interface{}, output *os.File) (string, error) {
	inst, ok := source.(*al64.ALInstrument)

	if !ok {
		return "", errors.New("Expected ALInstrument")
	}

	var name = "." + string(filepath.Separator) + "instruments" + string(filepath.Separator) + state.getUniqueName(".sfz")

	var filename = filepath.Join(state.cwd, name)

	audioconvert.EnsureDirectory(filename)

	instFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)

	if err != nil {
		return "", err
	}

	defer instFile.Close()

	for _, sound := range inst.SoundArray {
		instFile.WriteString("\n<region>\n")

		soundName, err := state.writeSection(sound.Wavetable, instFile, state.nameHint+"Snd", writeSfzWavetable)

		if err != nil {
			return "", err
		}

		var sfzPan = (float64(inst.Pan) + float64(sound.SamplePan) - 128) / 128

		if sfzPan > 1 {
			sfzPan = 1
		} else if sfzPan < -1 {
			sfzPan = -1
		}

		if sfzPan != 0 {
			instFile.WriteString(fmt.Sprintf("pan=%.06f\n", sfzPan*100))
		}

		instFile.WriteString(fmt.Sprintf("sample=.%s\n", soundName))

		var volumeScale = float64(int(inst.Volume)*int(sound.SampleVolume)) / (127 * 127)

		if sound.Envelope != nil {
			volumeScale *= float64(sound.Envelope.AttackVolume) / 127
		}

		if volumeScale < 1 {
			if volumeScale == 0 {
				instFile.WriteString(fmt.Sprintf("volume=-144\n"))
			} else {
				instFile.WriteString(fmt.Sprintf("volume=%.06f\n", math.Log(volumeScale)/math.Log(1.071773463)))
			}
		}

		writeSfzKeyMap(sound.KeyMap, instFile)
		writeSfzEnvelope(sound.Envelope, instFile)

		if sound.Wavetable.AdpcWave.Loop != nil {
			writeSfzLoop(sound.Wavetable.AdpcWave.Loop.Start, sound.Wavetable.AdpcWave.Loop.End, instFile)
		} else if sound.Wavetable.RawWave.Loop != nil {
			writeSfzLoop(sound.Wavetable.RawWave.Loop.Start, sound.Wavetable.RawWave.Loop.End, instFile)
		}
	}

	return name, nil
}

func writeSfzBank(state *insConversionState, source interface{}, output *os.File) (string, error) {
	var name = state.getUniqueName("")

	alBank, ok := source.(*al64.ALBank)

	if !ok {
		return name, errors.New("Expected ALBank")
	}

	state.sampleRate = alBank.SampleRate

	output.WriteString("\n<bank>\n")

	if alBank.Percussion != nil {
		percussionName, err := state.writeSection(alBank.Percussion, output, "Percussion", writeSfzInstrument)

		if err != nil {
			return name, err
		}

		output.WriteString("\n<percussion>\n")
		output.WriteString(fmt.Sprintf("instrument=%s\n", percussionName))
	}

	for index, instrument := range alBank.InstArray {
		if instrument != nil {
			percussionName, err := state.writeSection(instrument, output, MIDINames[index], writeSfzInstrument)

			if err != nil {
				return name, err
			}

			output.WriteString("\n<instrument>\n")
			output.WriteString(fmt.Sprintf("program_number=%d\n", index+1))
			output.WriteString(fmt.Sprintf("instrument=%s\n", percussionName))
		}
	}

	return name, nil
}

func writeSfzBankFile(state *insConversionState, source interface{}, output *os.File) (string, error) {
	alBankFile, ok := source.(*al64.ALBankFile)

	if !ok {
		return "", errors.New("Expected ALBankFile")
	}

	output.WriteString("// This isn't a real szf file. It is for creating n64 instrument banks\n")

	for _, alBank := range alBankFile.BankArray {
		_, err := state.writeSection(alBank, output, state.nameHint, writeSfzBank)

		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func WriteSfzFile(albank *al64.ALBankFile, tblData []byte, filename string) error {
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

	_, err = state.writeSection(albank, file, nameHint, writeSfzBankFile)

	return err
}
