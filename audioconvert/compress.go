package audioconvert

import (
	"encoding/binary"
	"os"
	"path/filepath"

	"github.com/lambertjamesd/sfz2n64/adpcm"
	"github.com/lambertjamesd/sfz2n64/al64"
)

func Compress(wavetable *al64.ALWavetable, codebook *adpcm.Codebook) {
	if wavetable.Type == al64.AL_RAW16_WAVE {
		var adpcmLoop *adpcm.Loop = nil

		if wavetable.RawWave.Loop != nil {
			adpcmLoop = &adpcm.Loop{
				Start: int(wavetable.RawWave.Loop.Start),
				End:   int(wavetable.RawWave.Loop.End),
				Count: int(wavetable.RawWave.Loop.Count),
				State: [16]int16{},
			}
		}

		result := adpcm.EncodeADPCM(
			&adpcm.PCMEncodedData{Samples: DecodeSamples(wavetable.DataFromTable, binary.BigEndian)},
			codebook,
			adpcmLoop,
			false,
			16,
		)

		wavetable.DataFromTable = adpcm.EnocdeFrames(result.Frames)
		wavetable.Len = int32(len(wavetable.DataFromTable))
		wavetable.Type = al64.AL_ADPCM_WAVE

		wavetable.AdpcWave.Book = ConvertCodebookToAL64(result.Codebook)

		if result.Loop != nil {
			wavetable.AdpcWave.Loop = &al64.ALADPCMloop{
				Start: uint32(result.Loop.Start),
				End:   uint32(result.Loop.End),
				Count: uint32(result.Loop.Count),
				State: result.Loop.State,
			}
		}

		wavetable.RawWave.Loop = nil
	}
}

func CompressWithSettings(wavetable *al64.ALWavetable, fileLocation string, compressionSettings *adpcm.CompressionSettings) error {
	if wavetable.Type != al64.AL_RAW16_WAVE {
		return nil
	}

	var existingTable = fileLocation[0:len(fileLocation)-len(filepath.Ext(fileLocation))] + ".table"

	var codebook *adpcm.Codebook

	if _, err := os.Stat(existingTable); err == nil {
		file, err := os.Open(existingTable)

		if err != nil {
			return err
		}

		defer file.Close()

		codebook, err = adpcm.ParseCodebook(file)

		if err != nil {
			return err
		}
	} else {
		codebook, err = adpcm.CalculateCodebook(
			DecodeSamples(wavetable.DataFromTable, binary.BigEndian),
			compressionSettings,
		)

		if err != nil {
			return err
		}
	}

	Compress(wavetable, codebook)

	return nil
}
