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

	err = reader.AddMetadata("Date", "2002-12-02")
	err = reader.AddMetadata("Artist", "Test Artist")
	err = reader.AddMetadata("Album", "Test Album")

	if err != nil {
		panic(err)
	}

	err = reader.Save("with_metadata.flac")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
