package main

import (
	"fmt"

	flacgo "github.com/jacopo-degattis/flacgo"
)

func main() {
	reader, err := flacgo.Open("examples/sample.flac")

	if err != nil {
		panic(err)
	}

	err = reader.RemoveMetadata("album", false)
	err = reader.RemoveMetadata("artist", false)

	if err != nil {
		panic(err)
	}

	err = reader.Save("with_metadata.flac")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
