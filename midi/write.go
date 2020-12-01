package midi

import (
	"bytes"
	"encoding/binary"
	"io"
)

func writeVarInt(writer io.Writer, value uint32, hasMore bool) error {
	var curr uint8 = uint8(value) & 0x7f

	value = value >> 7

	if value != 0 {
		writeVarInt(writer, value, true)
	}
	if hasMore {
		curr = curr | 0x80
	}
	return binary.Write(writer, binary.BigEndian, &curr)
}

func writeEvent(writer io.Writer, event *MidiEvent, prevEvent *MidiEvent) error {
	var delta uint32 = event.AbsoluteTime
	var err error

	if prevEvent != nil {
		delta = delta - prevEvent.AbsoluteTime
	}

	err = writeVarInt(writer, delta, false)

	if err != nil {
		return nil
	}

	if event.EventType == Metadata {
		var channelType uint8 = 0xFF

		err = binary.Write(writer, binary.BigEndian, &channelType)

		if err != nil {
			return err
		}

		err = binary.Write(writer, binary.BigEndian, &event.FirstParam)

		if err != nil {
			return err
		}

		var dataLen uint32 = uint32(len(event.Metadata))

		err = writeVarInt(writer, dataLen, false)

		if err != nil {
			return err
		}

		_, err = writer.Write(event.Metadata)

		if err != nil {
			return err
		}
	} else {
		if prevEvent == nil ||
			event.EventType != prevEvent.EventType ||
			event.Channel != prevEvent.Channel {
			var channelType uint8 = (uint8(event.EventType) << 4) | event.Channel

			err = binary.Write(writer, binary.BigEndian, &channelType)

			if err != nil {
				return err
			}
		}

		err = binary.Write(writer, binary.BigEndian, &event.FirstParam)

		if err != nil {
			return err
		}

		if bytesForEvent(event.EventType) == 2 {
			err = binary.Write(writer, binary.BigEndian, &event.SecondParam)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTrack(writer io.Writer, track *Track) error {
	var trackHeader uint32 = TrackHeader
	var err = binary.Write(writer, binary.BigEndian, &trackHeader)

	if err != nil {
		return err
	}

	var trackContent bytes.Buffer
	var prevEvent *MidiEvent = nil

	for _, event := range track.Events {
		writeEvent(&trackContent, event, prevEvent)
		prevEvent = event
	}

	var trackLength uint32 = uint32(trackContent.Len())

	err = binary.Write(writer, binary.BigEndian, &trackLength)

	if err != nil {
		return err
	}

	_, err = writer.Write(trackContent.Bytes())

	if err != nil {
		return err
	}

	return nil
}

func WriteMidi(writer io.Writer, midi *Midi) error {
	var header uint32 = MidiHeader
	var err = binary.Write(writer, binary.BigEndian, &header)

	if err != nil {
		return err
	}

	var length uint32 = 6
	err = binary.Write(writer, binary.BigEndian, &length)

	if err != nil {
		return err
	}

	var fileType uint16 = uint16(midi.Type)
	err = binary.Write(writer, binary.BigEndian, &fileType)

	if err != nil {
		return err
	}

	var trackCount uint16 = uint16(len(midi.Tracks))
	err = binary.Write(writer, binary.BigEndian, &trackCount)

	if err != nil {
		return err
	}

	err = binary.Write(writer, binary.BigEndian, &midi.TicksPerQuarter)

	if err != nil {
		return err
	}

	for _, track := range midi.Tracks {
		err = writeTrack(writer, track)

		if err != nil {
			return err
		}
	}

	return nil
}
