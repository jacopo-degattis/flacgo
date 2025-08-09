package main

import (
	"fmt"

	flacgo "github.com/jacopo-degattis/flacgo"
)

func main() {
	reader, err := flacgo.Open("examples/samplewithmetadata.flac")

	if err != nil {
		panic(err)
	}

	err = reader.RemoveMetadata("artist", false)

	if err != nil {
		panic(err)
	}

	outputPath := "without_artist.flac"
	err = reader.Save(&outputPath)

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
