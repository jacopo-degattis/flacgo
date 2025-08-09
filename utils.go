package flacgo

import (
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"
)

// GetAsText tries to parse an array of bytes as a string
func GetAsText(array []byte) string {
	var outputString string
	for _, integer := range array {
		outputString += string(integer)
	}
	return outputString
}

// ToBytes create a byte buffer of Uint32 values in the given ByteOrder
func ToBytes(value uint32, bytesLength int, endian binary.ByteOrder) []byte {
	tmpBuffer := make([]byte, 4)
	endian.PutUint32(tmpBuffer, value)
	return tmpBuffer[4-bytesLength:]
}

// GetCommentsLengthIndex returns the current number of vorbis comments and the index the value is stored at
func GetCommentsLengthIndex(vorbisBlock []byte) (uint32, uint32) {
	vendorLength := binary.LittleEndian.Uint32(vorbisBlock[0:4])
	numberOfComments := binary.LittleEndian.Uint32(vorbisBlock[4+vendorLength : 4+4+vendorLength])
	return numberOfComments, 4 + vendorLength
}

// AppendTo is like append but supports multiple []byte consequently
func AppendTo(slice []byte, elems [][]byte) []byte {
	for _, el := range elems {
		slice = append(slice, el...)
	}
	return slice
}

// containsIgnoreCase check if a slice contains or not the given string
func containsIgnoreCase(slice []string, str string) bool {
	for _, item := range slice {
		if strings.EqualFold(item, str) {
			return true
		}
	}
	return false
}

// GetFilteredBlocks filters out all the blocks provided as second parameter to the function
func GetFilteredBlocks(blocks []MetadataBlock, blockTypes []string) []MetadataBlock {
	filteredBlocks := make([]MetadataBlock, 0)
	for _, b := range blocks {
		if !containsIgnoreCase(blockTypes, b.BlockType) {
			filteredBlocks = append(filteredBlocks, b)
		}
	}
	return filteredBlocks
}

// IncreaseCommentsCounter increase the current vorbis comments counter of the given amount value
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

// For now support only for JPEG and PNG images
func ParseImage(filePath string) (image.Image, string, error) {
	f, err := os.Open(filePath)

	if err != nil {
		return nil, "", fmt.Errorf("unable to parse image: %w", err)
	}
	defer f.Close()

	image, imageType, err := image.Decode(f)

	return image, imageType, err
}
