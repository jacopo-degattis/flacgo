package flacgo

import (
	"encoding/binary"
	"fmt"
	"strings"
)

func GetAsText(array []byte) string {
	var outputString string
	for _, integer := range array {
		outputString += string(integer)
	}
	return outputString
}

func ToBytes(value uint32, bytesLength int, endian binary.ByteOrder) []byte {
	tmpBuffer := make([]byte, 4)
	endian.PutUint32(tmpBuffer, value)
	return tmpBuffer[4-bytesLength:]
}

// Get comments length from a given VORBIS_COMMENT block
// Returns: current number of comments and the byte index where the number is stored
func GetCommentsLengthIndex(vorbisBlock []byte) (uint32, uint32) {
	vendorLength := binary.LittleEndian.Uint32(vorbisBlock[0:4])
	numberOfComments := binary.LittleEndian.Uint32(vorbisBlock[4+vendorLength : 4+4+vendorLength])
	return numberOfComments, 4 + vendorLength
}

func AppendTo(slice []byte, elems [][]byte) []byte {
	for _, el := range elems {
		slice = append(slice, el...)
	}
	return slice
}

func GetVorbisBlock(flac *Flac) (*MetadataBlock, error) {
	metadataBlocks, err := flac.ReadAllMetadataBlocks()

	if err != nil {
		return nil, fmt.Errorf("unable to read all metadata blocks %w", err)
	}

	for _, block := range metadataBlocks {
		if block.BlockType == "VORBIS_COMMENT" {
			return &block, nil
		}
	}

	return nil, fmt.Errorf("unable to find vorbis metadata block")
}

func IncreaseCommentsCounter(fileBinary []byte, vorbisBlock []byte, amount int) {
	currentTotalComments, totalCommentsIndex := GetCommentsLengthIndex(vorbisBlock)
	updatedValue := ToBytes(currentTotalComments+1, 4, binary.LittleEndian)
	copy(fileBinary[totalCommentsIndex:totalCommentsIndex+4], updatedValue)
}

// If a duplicate exists this function will return the newComment value instead of the old one in order
// to replace the previous value with the new one and avoid duplicate metadata inside the vorbis block
func FilterDuplicatedComments(previousComments []VorbisComment, newComments []VorbisComment, removedComments map[string]bool) []VorbisComment {
	comments := make(map[string]VorbisComment)

	for _, oldComment := range previousComments {
		title := strings.ToLower(oldComment.Title)
		if !removedComments[title] {
			comments[title] = oldComment
		}
	}

	for _, newComment := range newComments {
		comments[strings.ToLower(newComment.Title)] = newComment
	}

	merged := make([]VorbisComment, 0, len(comments))
	for _, c := range comments {
		merged = append(merged, c)
	}

	return merged
}
