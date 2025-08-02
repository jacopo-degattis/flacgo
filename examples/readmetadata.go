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

	title, err := reader.ReadMetadata("title")
	artist, err := reader.ReadMetadata("artist")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] Flac track title: ", *title)
	fmt.Println("[+] Flac track artist: ", *artist)
}
