package convert

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/midi"
)

const percussionChannel = -1

func getUsedInstrument(bank *al64.ALBank, instrumentNumber int, key uint8, velocity uint8) (*al64.ALInstrument, *al64.ALSound) {
	var instrument *al64.ALInstrument = nil

	if instrumentNumber == percussionChannel {
		instrument = bank.Percussion
	} else {
		if instrumentNumber < len(bank.InstArray) {
			instrument = bank.InstArray[instrumentNumber]
		}
	}

	if instrument == nil {
		return nil, nil
	}

	for _, sound := range instrument.SoundArray {
		if sound.KeyMap.KeyMin <= key && sound.KeyMap.KeyMax >= key &&
			sound.KeyMap.VelocityMin <= velocity && sound.KeyMap.VelocityMax >= velocity {
			return instrument, sound
		}
	}

	return nil, nil
}

func markUsedSounds(bank *al64.ALBank, seqArray []*midi.Midi, into map[interface{}]bool) error {
	var programs [16]int

	programs[9] = percussionChannel

	for _, seq := range seqArray {
		for _, track := range seq.Tracks {
			for _, event := range track.Events {
				if event.EventType == midi.ProgramChange {
					programs[event.Channel] = int(event.FirstParam)
				} else if event.EventType == midi.MidiOn {
					inst, sound := getUsedInstrument(bank, programs[event.Channel], event.FirstParam, event.SecondParam)

					if inst == nil || sound == nil {
						return errors.New(fmt.Sprintf("Could not find instrument mapped to key=%d vel=%d instrument=%d", event.FirstParam, event.SecondParam, programs[event.Channel]))
					}

					into[inst] = true
					into[sound] = true
				}
			}
		}
	}

	return nil
}

func removeUnusedSoundsFromInstrument(instrument *al64.ALInstrument, used map[interface{}]bool) *al64.ALInstrument {
	var result al64.ALInstrument = *instrument

	result.SoundArray = nil

	for _, sound := range instrument.SoundArray {
		_, hasSound := used[sound]

		if hasSound {
			result.SoundArray = append(result.SoundArray, sound)
		}
	}

	return &result
}

func RemoveUnusedSounds(bank *al64.ALBank, seqArray []*midi.Midi) (*al64.ALBank, error) {
	var used = make(map[interface{}]bool)
	err := markUsedSounds(bank, seqArray, used)

	if err != nil {
		return nil, err
	}

	var result al64.ALBank

	result.SampleRate = bank.SampleRate

	if bank.Percussion != nil {
		_, instrumentUsed := used[bank.Percussion]

		if instrumentUsed {
			result.Percussion = removeUnusedSoundsFromInstrument(bank.Percussion, used)
		}
	}

	for _, inst := range bank.InstArray {
		_, instrumentUsed := used[inst]

		if instrumentUsed {
			result.InstArray = append(result.InstArray, removeUnusedSoundsFromInstrument(inst, used))
		} else {
			result.InstArray = append(result.InstArray, nil)
		}
	}

	return &result, nil
}

func ParseBankUsageFile(bankUsage string) ([][]*midi.Midi, error) {
	textData, err := ioutil.ReadFile(bankUsage)

	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(textData), "\n")

	var currBank = 0

	var result [][]*midi.Midi = nil

	for _, line := range lines {
		var trimmed = strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		asInt, err := strconv.ParseInt(trimmed, 10, 32)

		if err == nil {
			currBank = int(asInt)
			continue
		}

		midFile, err := os.Open(filepath.Join(filepath.Dir(bankUsage), trimmed))

		if err != nil {
			return nil, err
		}

		defer midFile.Close()

		midi, err := midi.ReadMidi(midFile)

		if err != nil {
			return nil, err
		}

		for currBank >= len(result) {
			result = append(result, nil)
		}

		result[currBank] = append(result[currBank], midi)
	}

	return result, nil
}
