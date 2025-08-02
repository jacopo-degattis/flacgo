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

	if err != nil {
		panic(err)
	}

	err = reader.AddCoverPicture("examples/test.jpg")

	if err != nil {
		panic(err)
	}

	err = reader.Save("with_picture.flac")

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
