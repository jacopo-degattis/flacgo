// Package flacgo provides a high-level library to work easily with
// FLAC files.
package flacgo

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
)

var BlockMapping = map[uint8]string{
	0:   "STREAMINFO",
	1:   "PADDING",
	2:   "APPLICATION",
	3:   "SEEKTABLE",
	4:   "VORBIS_COMMENT",
	5:   "CUESHEET",
	6:   "PICTURE",
	127: "INVALID",
}

// MetadataBlockHeader represents the header bytes of a MetadataBlock
type MetadataBlockHeader struct {
	BlockType   uint8
	IsLastBlock bool
	BlockLength uint32
}

// MetadataBlock stores all necessary informations about a MetadataBlock
type MetadataBlock struct {
	Index       int64
	BlockType   string
	IsLastBlock bool
	BlockHeader MetadataBlockHeader
	BlockData   []byte
}

// VorbisComment holds key and values to add a new VORBIS_COMMENT
type VorbisComment struct {
	Title string
	Value string
}

// Flac is the main struct holding a pointer to the currently opened file
type Flac struct {
	file *os.File
}

// Open a file from a given path
func Open(path string) (*Flac, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Failed to initialize flacgo %w", err)
	}

	// Check if the opened file is a valid FLAC file.
	magicHeader := make([]byte, 4)
	f.Read(magicHeader)
	if GetAsText(magicHeader) != "fLaC" {
		return nil, fmt.Errorf("Invalid FLAC format file.")
	}

	return &Flac{
		file: f,
	}, nil
}

// ReadBytes tries to read `bytesNum` amount of bytes from the currently open file
func (flac *Flac) ReadBytes(bytesNum int) ([]byte, error) {
	data := make([]byte, bytesNum)
	_, err := io.ReadFull(flac.file, data)

	if err != nil {
		return nil, fmt.Errorf("unable to read %d bytes from file: %w", bytesNum, err)
	}

	return data, nil
}

// ReadMetadataBlock reads a metadata block from the given offset
func (flac *Flac) ReadMetadataBlock(offset int64) (*MetadataBlock, error) {
	flac.file.Seek(offset, io.SeekStart)
	blockHeader, err := flac.ReadBytes(1)

	blockType := blockHeader[0] & 0x7F
	isLastBlock := (blockHeader[0] & 0x80) >> 7

	lengthBytes, err := flac.ReadBytes(3)

	// Uint32 requires 4 bytes slice to convert to Uint32 so I add one before as 0x00
	blockLength := binary.BigEndian.Uint32(append([]byte{0}, lengthBytes...))

	blockContent, err := flac.ReadBytes(int(blockLength))

	if err != nil {
		return nil, fmt.Errorf("unable to read metadata block with offset %d: %w", offset, err)
	}

	return &MetadataBlock{
		Index:       offset,
		BlockType:   BlockMapping[blockType],
		IsLastBlock: isLastBlock == 1,
		BlockHeader: MetadataBlockHeader{
			BlockType:   blockType,
			IsLastBlock: isLastBlock == 1,
			BlockLength: blockLength,
		},
		BlockData: blockContent,
	}, nil
}

// ReadAllMetadataBlocks tries to read all metadata blocks from a source FLAC file
func (flac *Flac) ReadAllMetadataBlocks() ([]MetadataBlock, error) {
	var offset int = 4
	blocks := []MetadataBlock{}

	for {
		data, err := flac.ReadMetadataBlock(int64(offset))
		if err != nil {
			return nil, fmt.Errorf("unable to read metadata block with offset %d: %w", offset, err)
		}

		offset += 4 + len(data.BlockData)
		blocks = append(blocks, *data)

		if data.IsLastBlock {
			break
		}
	}

	return blocks, nil
}

// ParseVorbisBlock tries to parse bytes from a vorbis block into a human readable structure
func (flac *Flac) ParseVorbisBlock(vorbisBlock []byte) ([]VorbisComment, error) {
	var vorbisComments []VorbisComment

	if len(vorbisBlock) < 8 {
		return nil, fmt.Errorf("vorbis block is too short")
	}

	vendorLength := binary.LittleEndian.Uint32(vorbisBlock[0:4])

	if len(vorbisBlock) < int(4+4+vendorLength) {
		return nil, fmt.Errorf("vorbis block too short for vendor length")
	}

	numberOfComments := binary.LittleEndian.Uint32(vorbisBlock[4+vendorLength : 4+4+vendorLength])

	iteration := 0
	offset := 4 + 4 + vendorLength
	for iteration < int(numberOfComments) {
		if len(vorbisBlock) < int(offset)+4 {
			return nil, fmt.Errorf("unexpected end of vorbis block while reading comment length")
		}

		commentLength := binary.LittleEndian.Uint32(vorbisBlock[offset : offset+4])

		if len(vorbisBlock) < int(offset)+int(commentLength) {
			return nil, fmt.Errorf("unexpected end of vorbis block while reading comment content")
		}

		commentContent := string(vorbisBlock[offset+4 : offset+4+commentLength])

		values := strings.Split(commentContent, "=")

		if len(values) != 2 {
			return nil, fmt.Errorf("malformed comment (no '=' found): %q", commentContent)
		}

		vorbisComments = append(vorbisComments, VorbisComment{
			Title: values[0],
			Value: values[1],
		})

		offset += commentLength + 4
		iteration += 1
	}

	return vorbisComments, nil
}

// CreateVorbisBlock creates a new VORBIS_COMMENT metadata block inside the flac file
func (flac *Flac) CreateVorbisBlock(fields map[string]string) ([]byte, error) {
	fmt.Print("[!] No vorbis block creating it...")

	blockType := 4 // 4 = VORBIS_COMMENT
	var lastBlockIndex int64
	allBlocks, err := flac.ReadAllMetadataBlocks()

	if err != nil {
		return nil, fmt.Errorf("unable to read metadata blocks %w", err)
	}

	for _, block := range allBlocks {
		if block.BlockType == "VORBIS_COMMENT" {
			// Can't add a new VORBIS_COMMENT then
			return nil, fmt.Errorf("You can have only one VORBIS_COMMENT per .flac file and there's one alredy.\n")
		}
		if block.IsLastBlock {
			lastBlockIndex = block.Index
		}
	}

	var body []byte
	vendor := []byte("flacgo1.1")
	vendorLength := ToBytes(uint32(len(vendor)), 4, binary.LittleEndian)
	newCommentsLength := ToBytes(uint32(len(fields)), 4, binary.LittleEndian)

	body = AppendTo(body, [][]byte{vendorLength, vendor, newCommentsLength})

	for key, value := range fields {
		comment := []byte(fmt.Sprintf("%s=%s", key, value))
		commentLength := ToBytes(uint32(len(comment)), 4, binary.LittleEndian)

		body = AppendTo(body, [][]byte{commentLength, comment})
	}

	isLast := 0
	payloadLength := ToBytes(uint32(len(body)), 3, binary.BigEndian)
	headerByte := (isLast << 7) | blockType

	header := []byte{byte(headerByte)}
	header = append(header, payloadLength...)

	fileInfo, err := flac.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to stat file: %w", err)
	}
	totalSize := fileInfo.Size()

	// prevData represnts file bytes from start to last block index
	flac.file.Seek(0, 0)
	prevData, err := flac.ReadBytes(int(lastBlockIndex))

	// postData represnts file bytes from last block index to file end
	flac.file.Seek(lastBlockIndex, 0)
	postData, err := flac.ReadBytes(int(totalSize - lastBlockIndex))

	if err != nil {
		return nil, fmt.Errorf("unable to read bytes from opened file %w", err)
	}

	newBuffer := AppendTo(prevData, [][]byte{header, body, postData})

	return newBuffer, nil
}

// AddMetadata simplify creating or appending new metadata inside the VORBIS_COMMENT metadata block
func (flac *Flac) AddMetadata(title string, value string) ([]byte, error) {
	allBlocks, err := flac.ReadAllMetadataBlocks()

	if err != nil {
		return nil, fmt.Errorf("unable to read all metadata blocks from file %w", err)
	}

	var vorbisBlockIndex int64 = 0
	// var vorbisHeader MetadataBlockHeader
	var vorbisBlockData []byte = []byte{}
	for _, block := range allBlocks {
		if block.BlockType == "VORBIS_COMMENT" {
			vorbisBlockIndex = block.Index
			vorbisBlockData = block.BlockData
			// vorbisHeader = block.BlockHeader
			break
		}
	}

	if vorbisBlockIndex == 0 {
		// It means that no VORBIS_COMMENT block currently exists
		vorbisBuffer, err := flac.CreateVorbisBlock(map[string]string{
			title: value,
		})

		if err != nil {
			return nil, fmt.Errorf("unable to create vorbis block %w", err)
		}

		return vorbisBuffer, nil
	}

	fmt.Print("[!] Vorbis block found udpating it...")
	// Create a copy of the current VORBIS_COMMENT block
	updatedBody := make([]byte, len(vorbisBlockData))
	copy(updatedBody, vorbisBlockData)

	// Else it means the block already exists and I just need to update
	// First I need to update the comments counter
	currentTotalComments, totalCommentsIndex := GetCommentsLengthIndex(vorbisBlockData)
	updatedValue := ToBytes(currentTotalComments+1, 4, binary.LittleEndian)
	copy(updatedBody[totalCommentsIndex:totalCommentsIndex+4], updatedValue)

	// Then create and encode new metadata infos
	newComment := []byte(fmt.Sprintf("%s=%s", title, value))
	newCommentLength := ToBytes(uint32(len(newComment)), 4, binary.LittleEndian)

	updatedBody = AppendTo(updatedBody, [][]byte{newCommentLength, newComment})

	fileInfo, err := flac.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("Failed to stat file: %w", err)
	}
	totalSize := fileInfo.Size()

	// prevData represnts file bytes from start to last block index
	flac.file.Seek(0, 0)
	prevData, err := flac.ReadBytes(int(vorbisBlockIndex))

	// postData represnts file bytes from last block index to file end
	// I need to calculate the size of the previous vorbisBlockData in order to read the file from that point on...
	originalBlockSize := int64(4 + len(vorbisBlockData))
	flac.file.Seek(vorbisBlockIndex+originalBlockSize, 0)
	postData, err := flac.ReadBytes(int(totalSize - (vorbisBlockIndex + originalBlockSize)))

	if err != nil {
		return nil, fmt.Errorf("unable to build new buffer after tagging %w", err)
	}

	// Rebuild the header
	blockSize := len(updatedBody) // The payload size in bytes
	payloadLength := []byte{
		byte((blockSize >> 16) & 0xFF),
		byte((blockSize >> 8) & 0xFF),
		byte(blockSize & 0xFF),
	}
	header := []byte{0x04}
	header = append(header, payloadLength...)

	newBuffer := AppendTo(prevData, [][]byte{header, updatedBody, postData})

	return newBuffer, nil
}
