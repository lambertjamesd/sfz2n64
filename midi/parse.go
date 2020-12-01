package midi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

func readVarInt(io io.Reader) (uint32, uint32, error) {
	var result uint32 = 0
	var bytesRead uint32 = 0
	var hasMore = true

	for hasMore {
		var readByte uint8
		err := binary.Read(io, binary.BigEndian, &readByte)
		bytesRead = bytesRead + 1

		if err != nil {
			return 0, 0, err
		}

		result = (result << 7) | (uint32(readByte) & 0x7F)

		hasMore = (readByte & 0x80) != 0
	}

	return result, bytesRead, nil
}

func readMidiEvent(reader io.Reader, prevEvent *MidiEvent) (*MidiEvent, uint32, error) {
	eventTime, bytesRead, err := readVarInt(reader)

	if prevEvent != nil {
		eventTime = eventTime + prevEvent.AbsoluteTime
	}

	if err != nil {
		return nil, 0, err
	}

	var eventChannel uint8

	err = binary.Read(reader, binary.BigEndian, &eventChannel)
	bytesRead = bytesRead + 1

	if err != nil {
		return nil, bytesRead, err
	}

	var channel uint8
	var eventType MidiEventType
	var firstByte uint8

	if eventChannel < 128 {
		if prevEvent == nil {
			return nil, 0, errors.New("Running event with no previous midi event")
		} else {
			channel = prevEvent.Channel
			eventType = prevEvent.EventType
		}

		firstByte = eventChannel
	} else {
		channel = eventChannel & 0xF
		eventType = MidiEventType(eventChannel >> 4)

		err = binary.Read(reader, binary.BigEndian, &firstByte)
		bytesRead = bytesRead + 1

		if err != nil {
			return nil, bytesRead, err
		}
	}

	if eventType == Metadata {
		extraBytes, extraByteLen, err := readVarInt(reader)
		bytesRead = bytesRead + extraByteLen

		if err != nil {
			return nil, bytesRead, err
		}

		if extraBytes == 0 {
			return &MidiEvent{
				eventTime,
				eventType,
				channel,
				firstByte,
				0,
				nil,
			}, bytesRead, nil
		} else {
			var data []byte = make([]byte, extraBytes)
			dataLength, err := reader.Read(data)
			bytesRead = bytesRead + uint32(extraBytes)

			if err != nil {
				return nil, bytesRead, err
			}

			if dataLength != int(extraBytes) {
				return nil, bytesRead, errors.New("Could not read data")
			}

			return &MidiEvent{
				eventTime,
				eventType,
				channel,
				firstByte,
				0,
				data,
			}, bytesRead, nil
		}
	} else {
		var secondByte uint8

		if firstByte&0x80 != 0 {
			return nil, bytesRead, errors.New("Data had high bit set")
		}

		if bytesForEvent(eventType) == 2 {
			err = binary.Read(reader, binary.BigEndian, &secondByte)
			bytesRead = bytesRead + 1

			if err != nil {
				return nil, bytesRead, err
			}

			if secondByte&0x80 != 0 {
				return nil, bytesRead, errors.New("Data had high bit set")
			}
		} else {
			secondByte = 0
		}

		return &MidiEvent{
			eventTime,
			eventType,
			channel,
			firstByte,
			secondByte,
			nil,
		}, bytesRead, nil
	}
}

func readTrack(reader io.Reader) (*Track, error) {
	var trackHeader uint32
	err := binary.Read(reader, binary.BigEndian, &trackHeader)

	if err != nil {
		return nil, err
	}

	if trackHeader != TrackHeader {
		return nil, errors.New("Invalid track header")
	}

	var trackLength uint32

	err = binary.Read(reader, binary.BigEndian, &trackLength)

	if err != nil {
		return nil, err
	}

	var bytesRead uint32 = 0
	var events []*MidiEvent = nil
	var prevEvent *MidiEvent = nil

	for bytesRead < trackLength {
		event, byteLength, err := readMidiEvent(reader, prevEvent)

		if err != nil {
			return nil, err
		}

		if event != nil {
			events = append(events, event)
			prevEvent = event
		}
		bytesRead = bytesRead + byteLength
	}

	return &Track{
		events,
	}, nil
}

func ReadMidi(reader io.Reader) (*Midi, error) {
	var midiHeader uint32
	err := binary.Read(reader, binary.BigEndian, &midiHeader)

	if err != nil {
		return nil, err
	}

	if midiHeader != MidiHeader {
		return nil, errors.New(fmt.Sprintf("Invalid midi header %X", midiHeader))
	}

	var headerLength uint32
	err = binary.Read(reader, binary.BigEndian, &headerLength)

	if headerLength != 6 {
		return nil, errors.New(fmt.Sprintf("Invalid midi header length %d", headerLength))
	}

	var midiType uint16
	var trackCount uint16
	var deltaTicksPerQuarter uint16

	err = binary.Read(reader, binary.BigEndian, &midiType)

	if err != nil {
		return nil, err
	}

	if midiType > MultipleTracksAsync {
		return nil, errors.New("Invalid midi type")
	}

	err = binary.Read(reader, binary.BigEndian, &trackCount)

	if err != nil {
		return nil, err
	}

	err = binary.Read(reader, binary.BigEndian, &deltaTicksPerQuarter)

	if err != nil {
		return nil, err
	}

	var tracks []*Track = nil

	var trackIndex uint16 = 0

	for trackIndex < trackCount {
		track, err := readTrack(reader)

		if err != nil {
			return nil, err
		}

		tracks = append(tracks, track)
		trackIndex = trackIndex + 1
	}

	return &Midi{
		MidiFileType(midiType),
		deltaTicksPerQuarter,
		tracks,
	}, nil
}
