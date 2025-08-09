package main

import (
	"fmt"

	flacgo "github.com/jacopo-degattis/flacgo"
)

func main() {
	reader, err := flacgo.Open("output.flac")

	if err != nil {
		panic(err)
	}

	err = reader.SetMetadata("Date", "2002-12-02")
	err = reader.SetMetadata("Artist", "Testone Artista")
	err = reader.SetMetadata("Album", "Testone Albumista")

	if err != nil {
		panic(err)
	}

	err = reader.Save(nil)

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
