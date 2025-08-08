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

	err = reader.SetMetadata("Date", "2002-12-02")
	err = reader.SetMetadata("Artist", "Test Artist")
	err = reader.SetMetadata("Album", "Test Album")

	if err != nil {
		panic(err)
	}

	err = reader.Save("output.flac")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
