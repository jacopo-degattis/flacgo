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

	header := reader.ReadBytes(4)

	if reader.GetAsText(header) != "fLaC" {
		panic("File is not in a valid .flac format")
	}

	metadataContent := reader.ReadMetadataBlock(4)

	fmt.Println("[=] Streaminfo data [=]")
	fmt.Println("[+] Block type ", metadataContent.BlockType)
	fmt.Println("[+] Is Last block ", metadataContent.IsLastBlock)
	fmt.Println("[+] Block Index", metadataContent.Index)
	fmt.Println("[+] Block content", metadataContent.BlockData)

	allBlocks := reader.ReadAllMetadataBlocks()

	fmt.Println()
	fmt.Printf("[+] Found a total of %d metadata blocks\n", len(allBlocks))

	for _, block := range allBlocks {
		if block.BlockType == "VORBIS_COMMENT" {
			infos := reader.ParseVorbisBlock(block.BlockData)

			fmt.Println()
			fmt.Println("[+] Vorbis block infos")
			fmt.Printf("[+] Block data: %s \n", infos)
		}
	}

	err = reader.CreateVorbisBlock(map[string]string{
		"Title":  "Prova",
		"Artist": "Test",
	})

	if err != nil {
		fmt.Printf("[-] Got error: %s", err)
		return
	}

	fmt.Println("[+] Metadata written to file golangtest.flac.")
}
