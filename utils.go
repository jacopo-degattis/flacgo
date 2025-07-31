package flacgo

import "encoding/binary"

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
