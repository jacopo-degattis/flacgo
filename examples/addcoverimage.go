package main

import (
	"fmt"
	"os"

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

	// You can choose to add picture either from PATH or from BYTES
	// err = reader.SetCoverPictureFromPath("examples/test.jpg")
	data, err := os.ReadFile("examples/test.jpg")

	if err != nil {
		panic(err)
	}

	err = reader.SetCoverPictureFromBytes(data)

	outputPath := "with_picture.flac"
	err = reader.Save(&outputPath)

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
