package midi

import (
	"fmt"
	"log"
)

const MidiHeader = 0x4D546864
const TrackHeader = 0x4D54726B

type MidiEventType uint8

const (
	MidiOff           = 0x8
	MidiOn            = 0x9 // FirstParam=noteNumber SecondParam=velocity
	AfterTouch        = 0xA
	ControlChange     = 0xB
	ProgramChange     = 0xC // FirstParam=program index
	ChannelAfterTouch = 0xD
	PitchWheel        = 0xE
	Metadata          = 0xF
)

type MetadataEventType uint8

const (
	MetaSequenceNumber = 0x00
	MetaText           = 0x01
	MetaCopyright      = 0x02
	MetaTrack          = 0x03
	MetaInstrument     = 0x04
	MetaLyric          = 0x05
	MetaMarker         = 0x06
	MetaCue            = 0x07
	MetaEnd            = 0x2F
	MetaTempo          = 0x51
	MetaTime           = 0x58
	MetaKey            = 0x59
	MetaSeqInfo        = 0x7F
)

type MidiEvent struct {
	AbsoluteTime uint32
	EventType    MidiEventType
	Channel      uint8
	FirstParam   uint8
	SecondParam  uint8
	Metadata     []byte
}

type Track struct {
	Events []*MidiEvent
}

type MidiFileType uint16

const (
	SingleTrack         = 0x0
	MultipleTracks      = 0x1
	MultipleTracksAsync = 0x2
)

type Midi struct {
	Type            MidiFileType
	TicksPerQuarter uint16
	Tracks          []*Track
}

func bytesForEvent(eventType MidiEventType) int {
	switch eventType {
	case MidiOff:
		return 2
	case MidiOn:
		return 2
	case AfterTouch:
		return 2
	case ControlChange:
		return 2
	case ProgramChange:
		return 1
	case ChannelAfterTouch:
		return 1
	case PitchWheel:
		return 2
	}

	log.Fatal(fmt.Sprintf("Unknown event type %d", eventType))

	return 0
}
