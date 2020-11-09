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
		velocityMin, velocityMax, err := sfzParseRange(key, lokey, hikey)

		if err != nil {
			return nil, err
		}

		keyMap.KeyMin, keyMap.KeyMax = velocityMin, velocityMax
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
		detune, err := strconv.ParseInt(pitch_keycenter, 10, 8)

		if err != nil {
			return nil, err
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

	attackTime, err := strconv.ParseFloat(attack, 64)

	if err != nil {
		return nil, err
	}

	result.AttackTime = int32(attackTime * 1000000)

	decayTime, err := strconv.ParseFloat(decay, 64)

	if err != nil {
		return nil, err
	}

	result.DecayTime = int32(decayTime * 1000000)

	releaseTime, err := strconv.ParseFloat(release, 64)

	if err != nil {
		return nil, err
	}

	result.ReleaseTime = int32(releaseTime * 1000000)

	decayVolume, err := strconv.ParseFloat(decay, 64)

	if decayVolume >= 100 {
		result.DecayVolume = 127
	} else if decayVolume < 0 {
		result.DecayVolume = 0
	} else {
		result.DecayVolume = uint8(decayVolume / 100 * 127)
	}

	return &result, nil
}

func sfzParseSound(region *sfz.SfzFullRegion) (*al64.ALSound, error) {
	filename := region.FindValue("sample")

	if filename == "" {
		return nil, errors.New("Region missing sample")
	}

	result, err := audioconvert.ReadWavetable(filename)

	keyMap, err := sfzParseKeyMap(region)

	if err != nil {
		return nil, err
	}

	result.KeyMap = keyMap

	env, err := sfzParseEnvelope(region)

	if err != nil {
		return nil, err
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

	return result, nil
}

func sfzParseInstrument(filename string) (*al64.ALInstrument, error) {
	sfzFile, err := sfz.ParseSfz(filename)

	if err != nil {
		return nil, err
	}

	var fullRegion *sfz.SfzFullRegion

	var instrument al64.ALInstrument

	for _, section := range sfzFile.Sections {
		if section.Name == "<global>" {
			fullRegion.Global = section
		} else if section.Name == "<group>" {
			fullRegion.Group = section
		} else if section.Name == "<region>" {
			fullRegion.Region = section
			sound, err := sfzParseSound(fullRegion)

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

func Sfz2N64(input *sfz.SfzFile, sfzFilename string) (*al64.ALBankFile, error) {
	var result al64.ALBankFile
	var currentBank *al64.ALBank

	for _, section := range input.Sections {
		if section.Name == "<bank>" {
			currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
		} else if section.Name == "<percussion>" {
			if currentBank == nil {
				currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
			}
			var instrumentName = section.FindValue("instrument")

			if instrumentName != "" {
				inst, err := sfzParseInstrument(filepath.Join(filepath.Dir(sfzFilename), instrumentName))

				if err != nil {
					return nil, err
				}

				currentBank.Percussion = inst
			} else {
				return nil, errors.New("<percussion> section defined without an instrument")
			}
		} else if section.Name == "<instrument>" {
			if currentBank == nil {
				currentBank = &al64.ALBank{SampleRate: 0, Percussion: nil, InstArray: nil}
			}

			var programNumberAsString = section.FindValue("program_number")
			var programNumber = 0

			if programNumberAsString != "" {
				parsedInt, err := strconv.ParseInt(programNumberAsString, 10, 32)

				if err != nil {
					return nil, errors.New(fmt.Sprintf("program_number should be a number not '%s'", programNumberAsString))
				}

				if parsedInt < 1 || parsedInt > 128 {
					return nil, errors.New(fmt.Sprintf("program_number should be a number between 1 and 128 not '%s'", programNumberAsString))
				}

				programNumber = int(parsedInt) - 1
			}

			for programNumber >= len(currentBank.InstArray) {
				currentBank.InstArray = append(currentBank.InstArray, nil)
			}

			var instrumentName = section.FindValue("instrument")

			if instrumentName == "" {
				return nil, errors.New("<instrument> section defined without an instrument")
			}

			inst, err := sfzParseInstrument(filepath.Join(filepath.Dir(sfzFilename), instrumentName))

			if err != nil {
				return nil, err
			}

			currentBank.InstArray[programNumber] = inst
		}
	}

	return &result, nil
}
