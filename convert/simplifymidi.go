package convert

import (
	"log"

	"github.com/lambertjamesd/sfz2n64/al64"
	"github.com/lambertjamesd/sfz2n64/midi"
)

type noteKey struct {
	instrument uint8
	node       uint8
}

func noteKeyFromMidi(programs *[16]int, event *midi.MidiEvent) noteKey {
	return noteKey{uint8(programs[event.Channel]), uint8(event.FirstParam)}
}

type activeNote struct {
	untilMicroseconds int
	currentSound      *al64.ALSound
	key               noteKey
}

type midiTime struct {
	ticksPerQuarter  int
	microsPerQuater  int
	currentMicroSecs int
	lastTick         int
}

const noNoteEnd int = int(uint(0xffffffffffffffff) >> 1)

func newMidiTime(ticksPerQuarter int) midiTime {
	return midiTime{
		ticksPerQuarter,
		500000,
		0,
		0,
	}
}

func (time *midiTime) updateTo(tick int) {
	var ticksPassed = tick - time.lastTick
	time.lastTick = tick
	time.currentMicroSecs = time.currentMicroSecs + ticksPassed*time.microsPerQuater/time.ticksPerQuarter
}

func removeStoppedSounds(noteMapping map[noteKey]*activeNote, microSeconds int) {
	var keysToRemove []noteKey = nil

	for note, sound := range noteMapping {
		if sound.untilMicroseconds < microSeconds {
			keysToRemove = append(keysToRemove, note)
		}
	}

	for _, note := range keysToRemove {
		delete(noteMapping, note)
	}
}

func SimplifyMidi(midiFile *midi.Midi, bank *al64.ALBank, maxActiveSounds int) int {
	var noteEndMapping = make(map[noteKey]*activeNote)
	var programs [16]int

	var maxActive = 0

	programs[9] = percussionChannel

	var time = newMidiTime(int(midiFile.TicksPerQuarter))

	for _, track := range midiFile.Tracks {
		for _, event := range track.Events {
			if event.EventType == midi.ProgramChange {
				programs[event.Channel] = int(event.FirstParam)
			} else if event.EventType == midi.MidiOn {
				time.updateTo(int(event.AbsoluteTime))
				removeStoppedSounds(noteEndMapping, time.currentMicroSecs)
				_, sound := getUsedInstrument(bank, programs[event.Channel], event.FirstParam, event.SecondParam)

				if sound != nil {
					var noteEndTime = noNoteEnd

					if sound.Envelope.DecayVolume == 0 {
						noteEndTime = time.currentMicroSecs + int(sound.Envelope.AttackTime+sound.Envelope.DecayTime)
					}

					var note = noteKeyFromMidi(&programs, event)
					noteEndMapping[note] = &activeNote{
						noteEndTime,
						sound,
						note,
					}

					if len(noteEndMapping) > maxActive {
						maxActive = len(noteEndMapping)
					}
				}
			} else if event.EventType == midi.MidiOff {
				time.updateTo(int(event.AbsoluteTime))

				var note = noteKeyFromMidi(&programs, event)

				active, has := noteEndMapping[note]

				if has {
					active.untilMicroseconds = time.currentMicroSecs + int(active.currentSound.Envelope.ReleaseTime)
				}
			} else if event.EventType == midi.Metadata && event.FirstParam == midi.MetaTempo {
				log.Fatal("Tempo midi event not currently supported\n")
			}
		}
	}

	return maxActive
}
