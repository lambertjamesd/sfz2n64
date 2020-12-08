package romextractor

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lambertjamesd/sfz2n64/al64"
)

type OffsetByteReader struct {
	content []byte
	offset  int
	curr    int
}

func (reader *OffsetByteReader) Read(p []byte) (n int, err error) {
	var actualRead = len(p)

	if actualRead > len(reader.content)-reader.curr {
		actualRead = len(reader.content) - reader.curr
	}

	for i := 0; i < actualRead; i++ {
		p[i] = reader.content[reader.curr+i]
	}

	reader.curr = reader.curr + actualRead

	if actualRead != len(p) {
		return actualRead, errors.New("Could not read full data")
	}

	return actualRead, nil
}

func (reader *OffsetByteReader) Seek(offset int64, whence int) (ret int64, err error) {
	if whence == os.SEEK_CUR {
		reader.curr = reader.curr + int(offset)
	} else if whence == os.SEEK_SET {
		reader.curr = reader.offset + int(offset)
	} else if whence == os.SEEK_END {
		reader.curr = len(reader.content) + int(offset)
	}

	if reader.curr < 0 {
		reader.curr = 0
	} else if reader.curr > len(reader.content) {
		reader.curr = len(reader.content)
	}

	return int64(reader.curr - reader.offset), nil
}

func FindBanks(content []byte) []*al64.ALBankFile {
	var result []*al64.ALBankFile = nil

	for i := 0; i < len(content); i++ {
		if content[i] == 0x42 && content[i+1] == 0x31 {
			// log.Println(fmt.Sprintf("Checking at offset %x\n", i))

			var reader = OffsetByteReader{
				content,
				i,
				i,
			}

			bankCheck, err := al64.ReadBankFile(&reader)

			if err == nil {
				log.Println(fmt.Sprintf("Found bank at offset %x\n", i))
				result = append(result, bankCheck)
			} else {
				log.Println(err.Error())
			}
		}
	}

	return result
}

func FindBanksInFile(filename string) ([]*al64.ALBankFile, error) {
	data, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	CorrectByteswap(data)

	result := FindBanks(data)

	return result, nil
}
