package convert

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strconv"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/audioconvert"
	"github.com/lambertjamesd/sfz2n64/sfz"
)

func sfzParseRange(absValueStr string, lowValueStr string, highValueStr string) (uint8, uint8, error) {
	var lowResult uint8 = 0
	var highResult uint8 = 127

	if absValueStr != "" {
		absValue, err := strconv.ParseInt(absValueStr, 10, 8)

		if err != nil {
			return 0, 0, err
		}

		lowResult = uint8(absValue)
		highResult = uint8(absValue)
	}

	if lowValueStr != "" {
		lowValue, err := strconv.ParseInt(lowValueStr, 10, 8)

		if err != nil {
			return 0, 0, err
		}

		lowResult = uint8(lowValue)
	}

	if highValueStr != "" {
		highValue, err := strconv.ParseInt(highValueStr, 10, 8)

		if err != nil {
			return 0, 0, err
		}

		highResult = uint8(highValue)
	}

	return lowResult, highResult, nil
}

func sfzParseKeyMap(region *sfz.SfzFullRegion) (*al64.ALKeyMap, error) {
	var keyMap al64.ALKeyMap

	key := region.FindValue("key")
	lokey := region.FindValue("lokey")
	hikey := region.FindValue("hikey")

	if key != "" || lokey != "" || hikey != "" {
		keyMin, keyMax, err := sfzParseRange(key, lokey, hikey)

		if err != nil {
			return nil, err
		}

		keyMap.KeyMin, keyMap.KeyMax = keyMin, keyMax
	} else {
		keyMap.KeyMin, keyMap.KeyMax = 0, 127
	}

	vel := region.FindValue("vel")
	lovel := region.FindValue("lovel")
	hivel := region.FindValue("hivel")

	if vel != "" || lovel != "" || hivel != "" {
		velocityMin, velocityMax, err := sfzParseRange(vel, lovel, hivel)

		if err != nil {
			return nil, err
		}

		keyMap.VelocityMin, keyMap.VelocityMax = velocityMin, velocityMax
	} else {
		keyMap.VelocityMin, keyMap.VelocityMax = 0, 127
	}

	pitch_keycenter := region.FindValue("pitch_keycenter")

	if pitch_keycenter == "" {
		keyMap.KeyBase = keyMap.KeyMin
	} else {
		keyBase, err := strconv.ParseInt(pitch_keycenter, 10, 8)

		if err != nil {
			return nil, err
		}

		keyMap.KeyBase = uint8(keyBase)
	}

	tune := region.FindValue("tune")

	if tune == "" {
		keyMap.Detune = 0
	} else {
		detune, err := strconv.ParseInt(tune, 10, 32)

		if err != nil {
			return nil, err
		}

		for detune > 50 {
			detune -= 100
			keyMap.KeyBase--
		}

		for detune < -50 {
			detune += 100
			keyMap.KeyBase++
		}

		keyMap.Detune = uint8(detune)
	}

	return &keyMap, nil
}

func sfzParseEnvelope(region *sfz.SfzFullRegion) (*al64.ALEnvelope, error) {
	attack := region.FindValue("ampeg_attack")
	decay := region.FindValue("ampeg_decay")
	release := region.FindValue("ampeg_release")
	sustainLevel := region.FindValue("ampeg_sustain")

	if attack == "" && decay == "" && release == "" && sustainLevel == "" {
		return nil, nil
	}

	var result al64.ALEnvelope

	result.AttackVolume = 127

	var attackTime float64
	var decayTime float64
	var releaseTime float64
	var err error

	if attack != "" {
		attackTime, err = strconv.ParseFloat(attack, 64)

		if err != nil {
			return nil, errors.New("ampeg_attack should be a number")
		}
	}

	result.AttackTime = int32(attackTime * 1000000)

	if decay != "" {
		decayTime, err = strconv.ParseFloat(decay, 64)

		if err != nil {
			return nil, errors.New("ampeg_decay should be a number")
		}
	}

	result.DecayTime = int32(decayTime * 1000000)

	if release != "" {
		releaseTime, err = strconv.ParseFloat(release, 64)

		if err != nil {
			return nil, errors.New("ampeg_attack should be a number")
		}
	}

	result.ReleaseTime = int32(releaseTime * 1000000)

	var decayVolume float64

	if sustainLevel != "" {
		decayVolume, err = strconv.ParseFloat(sustainLevel, 64)

		if err != nil {
			return nil, errors.New("ampeg_sustain should be a number")
		}
	} else {
		decayVolume = 100
	}

	if decayVolume >= 100 {
		result.DecayVolume = 127
	} else if decayVolume < 0 {
		result.DecayVolume = 0
	} else {
		result.DecayVolume = uint8(decayVolume / 100 * 127)
	}

	return &result, nil
}

func sfzParseLoop(region *sfz.SfzFullRegion, sound *al64.ALSound) error {
	var loopMode = region.FindValue("loop_mode")
	var loopStart = region.FindValue("loop_start")
	var loopEnd = region.FindValue("loop_end")

	if loopMode != "" || loopStart != "" || loopEnd != "" {
		var start uint32
		var end uint32

		if loopStart == "" {
			start = 0
		} else {
			start64, err := strconv.ParseInt(loopStart, 10, 32)

			if err != nil {
				return errors.New("Invalid value for loop_start")
			}

			start = uint32(start64)
		}

		if loopEnd == "" {
			end = 0
		} else {
			end64, err := strconv.ParseInt(loopEnd, 10, 32)

			if err != nil {
				return errors.New("Invalid value for loop_end")
			}

			end = uint32(end64)
		}

		sound.Wavetable.RawWave.Loop = &al64.ALRawLoop{
			Start: start,
			End:   end + 1,
			Count: ^uint32(0),
		}
	}

	return nil
}

func sfzParseSound(region *sfz.SfzFullRegion) (*al64.ALSound, error) {
	filename := region.FindValue("sample")

	if filename == "" {
		return nil, errors.New("Region missing sample")
	}

	result, err := audioconvert.ReadWavetable(filename)

	if err != nil {
		return nil, err
	}

	keyMap, err := sfzParseKeyMap(region)

	if err != nil {
		return nil, err
	}

	result.KeyMap = keyMap

	env, err := sfzParseEnvelope(region)

	if err != nil {
		return nil, err
	}

	if env == nil {
		env = &al64.ALEnvelope{
			AttackTime:   0,
			DecayTime:    int32(1000000 * len(result.Wavetable.DataFromTable) / 2 / int(result.Wavetable.FileSampleRate)),
			ReleaseTime:  0,
			AttackVolume: 127,
			DecayVolume:  127,
		}
	}

	result.Envelope = env

	pan := region.FindValue("pan")

	if pan == "" {
		result.SamplePan = 64
	} else {
		panAsFloat, err := strconv.ParseFloat(pan, 64)

		if err != nil {
			return nil, err
		}

		if panAsFloat > 100 {
			result.SamplePan = 127
		} else if panAsFloat < -100 {
			result.SamplePan = 0
		} else {
			result.SamplePan = uint8((panAsFloat + 100) * 127 / 200)
		}
	}

	volume := region.FindValue("volume")

	if volume == "" {
		result.SampleVolume = 127
	} else {
		volumeAsFloat, err := strconv.ParseFloat(volume, 64)

		if err != nil {
			return nil, err
		}

		if volumeAsFloat >= 0 {
			result.SampleVolume = 127
		} else {
			var linearScale = math.Pow(1.071773463, volumeAsFloat)
			result.SampleVolume = uint8(linearScale * 127)
		}
	}

	sfzParseLoop(region, result)

	offset := region.FindValue("offset")

	var start = 0

	if offset != "" {
		offsetAsInt, err := strconv.ParseInt(offset, 10, 32)

		if err != nil {
			return nil, errors.New("offset should be number")
		}

		var data = result.Wavetable.DataFromTable
		data = data[offsetAsInt*2 : len(data)]
		result.Wavetable.DataFromTable = data

		start = int(offsetAsInt)

		if result.Wavetable.RawWave.Loop != nil {
			result.Wavetable.RawWave.Loop.Start -= uint32(offsetAsInt)
			result.Wavetable.RawWave.Loop.End -= uint32(offsetAsInt)
		}
	}

	end := region.FindValue("end")

	if end != "" {
		endAsInt, err := strconv.ParseInt(end, 10, 32)

		if err != nil {
			return nil, errors.New("end should be number")
		}

		var data = result.Wavetable.DataFromTable
		data = data[0 : (int(endAsInt)-start)*2]
		result.Wavetable.DataFromTable = data
	}

	result.Wavetable.Len = int32(len(result.Wavetable.DataFromTable))

	return result, nil
}

func sfzParseInstrument(sfzFile *sfz.SfzFile) (*al64.ALInstrument, error) {
	var fullRegion sfz.SfzFullRegion

	var instrument al64.ALInstrument

	for _, section := range sfzFile.Sections {
		if section.Name == "<global>" {
			fullRegion.Global = section
		} else if section.Name == "<group>" {
			fullRegion.Group = section
		} else if section.Name == "<region>" {
			fullRegion.Region = section
			sound, err := sfzParseSound(&fullRegion)

			if err != nil {
				return nil, err
			}

			instrument.SoundArray = append(instrument.SoundArray, sound)
		}
	}

	instrument.Volume = 127
	instrument.Pan = 64

	return &instrument, nil
}

func sfzParseInstrumentFile(filename string) (*al64.ALInstrument, error) {
	sfzFile, err := sfz.ParseSfz(filename)

	if err != nil {
		return nil, err
	}

	return sfzParseInstrument(sfzFile)
}

func SfzIsSingleInstrument(input *sfz.SfzFile) bool {
	for _, section := range input.Sections {
		if section.Name == "<bank>" || section.Name == "<percussion>" || section.Name == "<instrument>" {
			return false
		}
	}
	return true
}

func sfzParseAsBankFile(input *sfz.SfzFile, sfzFilename string) (*al64.ALBankFile, error) {
	var result al64.ALBankFile
	var currentBank *al64.ALBank

	var firstProgramIndex = 1

	for _, section := range input.Sections {
		if section.Name == "<bank>" {
			currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
			result.BankArray = append(result.BankArray, currentBank)

			var firstProgramIndexString = section.FindValue("first_program_index")

			if firstProgramIndexString != "" {
				parsedInt, err := strconv.ParseInt(firstProgramIndexString, 10, 32)

				if err != nil {
					return nil, errors.New("first_program_index must be a number")
				}

				firstProgramIndex = int(parsedInt)
			}
		} else if section.Name == "<percussion>" {
			if currentBank == nil {
				currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
				result.BankArray = append(result.BankArray, currentBank)
			}
			var instrumentName = section.FindValue("instrument")

			if instrumentName != "" {
				inst, err := sfzParseInstrumentFile(filepath.Join(filepath.Dir(sfzFilename), instrumentName))

				if err != nil {
					return nil, err
				}

				currentBank.Percussion = inst

				if len(inst.SoundArray) > 0 {
					currentBank.SampleRate = inst.SoundArray[0].Wavetable.FileSampleRate
				}
			} else {
				return nil, errors.New("<percussion> section defined without an instrument")
			}
		} else if section.Name == "<instrument>" {
			if currentBank == nil {
				currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
				result.BankArray = append(result.BankArray, currentBank)
			}

			var programNumberAsString = section.FindValue("program_number")
			var programNumber = 0

			if programNumberAsString != "" {
				parsedInt, err := strconv.ParseInt(programNumberAsString, 10, 32)

				if err != nil {
					return nil, errors.New(fmt.Sprintf("program_number should be a number not '%s'", programNumberAsString))
				}

				programNumber = int(parsedInt) - firstProgramIndex

				if parsedInt < 0 {
					return nil, errors.New(fmt.Sprintf(
						"program_number should be a number greater than %d not '%s'",
						firstProgramIndex,
						programNumberAsString,
					))
				}
			}

			for programNumber >= len(currentBank.InstArray) {
				currentBank.InstArray = append(currentBank.InstArray, nil)
			}

			var instrumentName = section.FindValue("instrument")

			if instrumentName == "" {
				return nil, errors.New("<instrument> section defined without an instrument")
			}

			inst, err := sfzParseInstrumentFile(filepath.Join(filepath.Dir(sfzFilename), instrumentName))

			if err != nil {
				return nil, err
			}

			currentBank.InstArray[programNumber] = inst

			if len(inst.SoundArray) > 0 {
				currentBank.SampleRate = inst.SoundArray[0].Wavetable.FileSampleRate
			}
		}
	}

	result.CorrectOverlap()

	return &result, nil
}

func sfzParseAsSingleInstrument(input *sfz.SfzFile) (*al64.ALBankFile, error) {
	var result al64.ALBankFile
	var currentBank *al64.ALBank
	currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
	result.BankArray = append(result.BankArray, currentBank)
	inst, err := sfzParseInstrument(input)

	if err != nil {
		return nil, err
	}

	currentBank.InstArray = make([]*al64.ALInstrument, 1)
	currentBank.InstArray[0] = inst

	if len(inst.SoundArray) > 0 {
		currentBank.SampleRate = inst.SoundArray[0].Wavetable.FileSampleRate
	}

	return &result, nil
}

func Sfz2N64(input *sfz.SfzFile, sfzFilename string) (*al64.ALBankFile, error) {
	if SfzIsSingleInstrument(input) {
		return sfzParseAsSingleInstrument(input)
	} else {
		return sfzParseAsBankFile(input, sfzFilename)
	}
}
