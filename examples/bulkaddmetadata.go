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

	data, err := os.ReadFile("examples/test.jpg")

	// You can either use this
	err = reader.BulkAddMetadata(flacgo.FlacMetadatas{
		Date:   "2002-12-02",
		Artist: "Test Artist",
		Album:  "Test Album",
		Cover:  data,
	})

	// Or this
	// err = reader.SetMetadata("Date", "2002-12-02")
	// err = reader.SetMetadata("Artist", "Test Artist")
	// err = reader.SetMetadata("Album", "Test Album")

	if err != nil {
		panic(err)
	}

	filename := "output.flac"
	err = reader.Save(&filename)

	if err != nil {
		panic(err)
	}

	fmt.Println("[+] DONE")
}
