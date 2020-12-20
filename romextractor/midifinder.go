package romextractor

import (
	"fmt"

	"github.com/lambertjamesd/sfz2n64/midi"
)

func FindMidi(content []byte) []*midi.Midi {
	var result []*midi.Midi = nil

	for i := 0; i < len(content); i++ {
		if content[i] == 0x4D && content[i+1] == 0x54 && content[i+2] == 0x68 && content[i+3] == 0x64 && (i & ^7) == i {
			var reader = OffsetByteReader{
				content,
				i,
				i,
			}

			midiCheck, err := midi.ReadMidi(&reader)

			if err == nil {
				fmt.Println(fmt.Sprintf("Found midi at offset %x", i))
				result = append(result, midiCheck)
			}
		}
	}

	return result
}
