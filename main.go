package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage sfz2n64 input.sfz output_prefix")
	}

	sfzFile, err := ParseSfz(os.Args[1])

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Parsed %s with %d sections", os.Args[1], len(sfzFile.Sections))
}
