package main

import (
	"os"

	flacgo "github.com/jacopo-degattis/flacgo"
)

func main() {
	reader, err := flacgo.Open("examples/sample.flac")

	if err != nil {
		panic(err)
	}

	newFileBuff, err := reader.AddMetadata("artist", "TEST_ARTIST")

	err = os.WriteFile("output_metadata.flac", newFileBuff, 0644)

	if err != nil {
		panic(err)
	}
}
