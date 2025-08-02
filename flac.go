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
	file            *os.File
	fileSize        int64
	vorbisIndex     int64
	vorbisLength    int
	pendingComments []VorbisComment
	parsedComments  []VorbisComment
	removedComments map[string]bool
}

// Open a file from a given path
func Open(path string) (*Flac, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize flacgo %w", err)
	}

	// Check if the opened file is a valid FLAC file.
	magicHeader := make([]byte, 4)
	f.Read(magicHeader)
	if GetAsText(magicHeader) != "fLaC" {
		return nil, fmt.Errorf("invalid FLAC format file.")
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to stat file %w", err)
	}

	flacRef := &Flac{
		file:     f,
		fileSize: fileInfo.Size(),
	}

	vorbisBlock, err := GetVorbisBlock(flacRef)
	flacRef.vorbisIndex = vorbisBlock.Index

	if err != nil {
		return nil, fmt.Errorf("unable to get comments from vorbis block %w", err)
	}

	parsedComments, err := flacRef.ParseVorbisBlock(vorbisBlock.BlockData)

	if err != nil {
		return nil, fmt.Errorf("unable to parse vorbis blocks %w", err)
	}

	flacRef.parsedComments = parsedComments
	flacRef.vorbisLength = int(vorbisBlock.BlockHeader.BlockLength)

	// Fill new comments to write with the parsed one, if no changes are made then it will write the same as before
	// NOTE: TODO: now the best because it write even tho is not necessary, fix??
	flacRef.pendingComments = parsedComments
	flacRef.removedComments = make(map[string]bool)

	return flacRef, nil
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
func (flac *Flac) CreateVorbisBlock() ([]byte, error) {
	blockType := 4 // 4 = VORBIS_COMMENT

	var body []byte
	vendor := []byte("flacgo1.1")
	vendorLength := ToBytes(uint32(len(vendor)), 4, binary.LittleEndian)

	allMetadata := FilterDuplicatedComments(flac.parsedComments, flac.pendingComments, flac.removedComments)
	newCommentsLength := ToBytes(uint32(len(allMetadata)), 4, binary.LittleEndian)
	body = AppendTo(body, [][]byte{vendorLength, vendor, newCommentsLength})

	fmt.Print(allMetadata)

	for _, cmt := range allMetadata {
		comment := []byte(fmt.Sprintf("%s=%s", cmt.Title, cmt.Value))
		commentLength := ToBytes(uint32(len(comment)), 4, binary.LittleEndian)

		body = AppendTo(body, [][]byte{commentLength, comment})
	}

	isLast := 0
	payloadLength := ToBytes(uint32(len(body)), 3, binary.BigEndian)
	headerByte := (isLast << 7) | blockType

	header := []byte{byte(headerByte)}
	header = append(header, payloadLength...)
	header = append(header, body...)

	return header, nil
}

// ReadMetadata from the currently open FLAC file
func (flac *Flac) ReadMetadata(title string) (*string, error) {
	for _, cmt := range flac.parsedComments {
		if strings.ToLower(cmt.Title) == title {
			return &cmt.Value, nil
		}
	}

	return nil, fmt.Errorf("no metadata found with title '%s'", title)
}

// AddMetadata inserts a new metadata inside the FLAC file only if the metadata doesn't already exists.
func (flac *Flac) AddMetadata(title string, value string) error {
	for _, cmt := range flac.parsedComments {
		if strings.ToLower(cmt.Title) == strings.ToLower(title) {
			return fmt.Errorf("unable to add metadata with name '%s' because it already exists.", title)
		}
	}

	flac.pendingComments = append(flac.pendingComments, VorbisComment{
		Title: title,
		Value: value,
	})

	return nil
}

// UpdateMetadata tries to update the value of a given metadata that already exists, if it doesn't exists then the function returns error.
func (flac *Flac) UpdateMetadata(title string, newValue string) error {
	for _, cmt := range flac.parsedComments {
		if strings.ToLower(cmt.Title) == strings.ToLower(title) {
			flac.pendingComments = append(flac.pendingComments, VorbisComment{
				Title: title,
				Value: newValue,
			})

			return nil
		}
	}

	return fmt.Errorf("unable to update metadata, metadata with name '%s' not found", title)
}

// SetMetadata inserts a new metadata inside the FLAC file, if it doesn't exists it creates it otherwise it updates the value.
func (flac *Flac) SetMetadata(title string, value string) error {
	flac.pendingComments = append(flac.pendingComments, VorbisComment{
		Title: title,
		Value: value,
	})

	return nil
}

// RemoveMetadata from the currently opened flac file.
// If IgnoreIfMissing is set to true then no error will be returned if the
// metadata key is missing.
func (flac *Flac) RemoveMetadata(title string, ignoreIfMissing bool) error {
	exists := false
	if !ignoreIfMissing {
		for _, cmt := range flac.parsedComments {
			if strings.ToLower(title) == strings.ToLower(cmt.Title) {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("unable to remove metadata with name '%s' because it doesn't exist in file.", title)
		}
	}

	updatedComments := make([]VorbisComment, 0)

	for _, cmt := range flac.pendingComments {
		if strings.ToLower(cmt.Title) != strings.ToLower(title) {
			updatedComments = append(updatedComments, cmt)
		}
	}

	fmt.Printf("Updated Comments: %s", updatedComments)
	flac.pendingComments = updatedComments
	flac.removedComments[strings.ToLower(title)] = true

	return nil
}

// Save the file locally.
func (flac *Flac) Save(path string) error {
	outFile, err := os.Create(path)

	if err != nil {
		return fmt.Errorf("unable to save file with name %s: %w", path, err)
	}
	defer outFile.Close()

	vorbisBody, err := flac.CreateVorbisBlock()
	if err != nil {
		return fmt.Errorf("unable to build new vorbis block: %w", err)
	}

	flac.file.Seek(0, 0)
	prevData, err := flac.ReadBytes(int(flac.vorbisIndex))

	originalBlockSize := int64(4 + flac.vorbisLength)

	flac.file.Seek(flac.vorbisIndex+originalBlockSize, 0)
	postData, err := flac.ReadBytes(int(flac.fileSize - (flac.vorbisIndex + originalBlockSize)))

	if err != nil {
		return fmt.Errorf("unable to read: %w", err)
	}

	outFile.Write(prevData)
	outFile.Write(vorbisBody)
	outFile.Write(postData)

	return nil
}
