package main

import (
	"fmt"

	flacgo "github.com/jacopo-degattis/flacgo"
)

func main() {
	reader, err := flacgo.Open("test.flac")

	if err != nil {
		panic(err)
	}

	reader.RemoveMetadata("album")
	reader.RemoveMetadata("artist")
	
	if err != nil {
		panic(err)
	}

	err = reader.Save("with_metadata.flac")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
