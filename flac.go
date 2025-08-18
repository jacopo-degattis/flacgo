// Package flacgo provides a high-level library to work easily with
// FLAC files.
package flacgo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
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
	Data        []byte
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

type FlacMetadatas struct {
	Title  string
	Artist string
	Album  string
	Date   string
	Cover  []byte
}

// Flac is the main struct holding a pointer to the currently opened file
type Flac struct {
	file                *os.File
	fileName            string
	fileSize            int64
	vorbisIndex         *int64
	vorbisLength        int
	pendingComments     []VorbisComment
	parsedComments      []VorbisComment
	removedComments     map[string]bool
	parsedCoverPicture  *MetadataBlock
	pendingCoverPicture []byte
	removeCoverPicture  bool
}

// Open a file from a given path
func Open(path string) (*Flac, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize flacgo: %w", err)
	}

	// Check if the opened file is a valid FLAC file.
	magicHeader := make([]byte, 4)
	f.Read(magicHeader)
	if GetAsText(magicHeader) != "fLaC" {
		return nil, fmt.Errorf("invalid FLAC format file, found '%s' instead", GetAsText(magicHeader))
	}

	fileInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to stat file %w", err)
	}

	flacRef := &Flac{
		file:               f,
		fileName:           f.Name(),
		fileSize:           fileInfo.Size(),
		removeCoverPicture: false,
	}

	vorbisBlock, _ := flacRef.getBlock("VORBIS_COMMENT")
	pictureBlock, _ := flacRef.getBlock("PICTURE")

	flacRef.parsedCoverPicture = pictureBlock

	if vorbisBlock == nil {
		flacRef.vorbisIndex = nil
		flacRef.parsedComments = make([]VorbisComment, 0)
		flacRef.pendingComments = make([]VorbisComment, 0)
		return flacRef, nil
	}

	flacRef.vorbisIndex = &vorbisBlock.Index
	flacRef.vorbisLength = int(vorbisBlock.BlockHeader.BlockLength)

	parsedComments, err := flacRef.parseVorbisBlock(vorbisBlock.BlockData)
	if err != nil {
		return nil, fmt.Errorf("unable to parse vorbis blocks %w", err)
	}

	flacRef.parsedComments = parsedComments
	// Fill new comments to write with the parsed one, if no changes are made then it will write the same as before
	// NOTE: TODO: now the best because it write even tho is not necessary, fix??
	flacRef.pendingComments = parsedComments
	flacRef.removedComments = make(map[string]bool)

	return flacRef, nil
}

// ReadBytes tries to read `bytesNum` amount of bytes from the currently open file
func (flac *Flac) readBytes(bytesNum int) ([]byte, error) {
	data := make([]byte, bytesNum)
	_, err := io.ReadFull(flac.file, data)

	if err != nil {
		return nil, fmt.Errorf("unable to read %d bytes from file: %w", bytesNum, err)
	}

	return data, nil
}

// GetBlock returns a MetadataBlock pointer of the requested block
func (flac *Flac) getBlock(blockType string) (*MetadataBlock, error) {
	isValidBlockType := false
	for _, v := range BlockMapping {
		if strings.EqualFold(v, blockType) {
			isValidBlockType = true
		}
	}
	if !isValidBlockType {
		return nil, fmt.Errorf("'%s' is an invalid block type", blockType)
	}

	var fullBlock *MetadataBlock = nil
	allBlocks, err := flac.readAllMetadataBlocks()

	if err != nil {
		return nil, fmt.Errorf("unable to read all metadata blocks: %w", err)
	}

	for _, block := range allBlocks {
		if strings.EqualFold(block.BlockType, blockType) {
			fullBlock = &block
		}
	}

	return fullBlock, nil
}

// ReadMetadataBlock reads a metadata block from the given offset
func (flac *Flac) readMetadataBlock(offset int64) (*MetadataBlock, error) {
	flac.file.Seek(offset, io.SeekStart)
	headerBytes, err := flac.readBytes(4)

	if err != nil {
		return nil, fmt.Errorf("unable to read header bytes from offset '%d': %w", offset, err)
	}

	lengthBytes := headerBytes[1:4]
	blockType := headerBytes[0] & 0x7F
	isLastBlock := (headerBytes[0] & 0x80) >> 7

	// Uint32 requires 4 bytes slice to convert to Uint32 so I add one before as 0x00
	blockLength := binary.BigEndian.Uint32(append([]byte{0}, lengthBytes...))

	blockContent, err := flac.readBytes(int(blockLength))

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
			Data:        headerBytes,
		},
		BlockData: blockContent,
	}, nil
}

// ReadAllMetadataBlocks tries to read all metadata blocks from a source FLAC file
func (flac *Flac) readAllMetadataBlocks() ([]MetadataBlock, error) {
	var offset int = 4
	blocks := []MetadataBlock{}

	for {
		data, err := flac.readMetadataBlock(int64(offset))
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
func (flac *Flac) parseVorbisBlock(vorbisBlock []byte) ([]VorbisComment, error) {
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

func (flac *Flac) parseImageFromPath(imagePath string) (*string, []byte, error) {
	_, imageType, err := ParseImage(imagePath)

	if err != nil {
		return nil, nil, fmt.Errorf("unable to read image: %w", err)
	}

	pictureMimeType := "image/" + imageType
	imageData, err := os.ReadFile(imagePath)

	if err != nil {
		return nil, nil, fmt.Errorf("unable to read image data %w", err)
	}

	return &pictureMimeType, imageData, nil
}

func (flac *Flac) createPictureBlock(imageData []byte, pictureMimeType string) ([]byte, error) {
	var buf bytes.Buffer
	var fullBuf bytes.Buffer

	header := byte(6)
	header |= 0x80
	fullBuf.WriteByte(header)

	binary.Write(&buf, binary.BigEndian, uint32(3))
	binary.Write(&buf, binary.BigEndian, uint32(len(pictureMimeType)))
	buf.WriteString(pictureMimeType)
	binary.Write(&buf, binary.BigEndian, uint32(len("")))
	buf.WriteString("")
	binary.Write(&buf, binary.BigEndian, uint32(600))
	binary.Write(&buf, binary.BigEndian, uint32(600))
	binary.Write(&buf, binary.BigEndian, uint32(24))
	binary.Write(&buf, binary.BigEndian, uint32(0))
	binary.Write(&buf, binary.BigEndian, uint32(len(imageData)))
	buf.Write(imageData)

	blockData := buf.Bytes()
	length := uint32(len(blockData))

	fullBuf.Write([]byte{
		byte((length >> 16) & 0xFF),
		byte((length >> 8) & 0xFF),
		byte(length & 0xFF),
	})
	fullBuf.Write(blockData)

	return fullBuf.Bytes(), nil
}

// CreateVorbisBlock creates a new VORBIS_COMMENT metadata block inside the flac file
func (flac *Flac) createVorbisBlock() ([]byte, error) {

	if len(flac.pendingComments) == 0 {
		return nil, nil
	}

	blockType := 4 // 4 = VORBIS_COMMENT

	var body []byte
	vendor := []byte("flacgo1.1")
	vendorLength := ToBytes(uint32(len(vendor)), 4, binary.LittleEndian)

	allMetadata := FilterDuplicatedComments(flac.parsedComments, flac.pendingComments, flac.removedComments)

	newCommentsLength := ToBytes(uint32(len(allMetadata)), 4, binary.LittleEndian)
	body = AppendTo(body, [][]byte{vendorLength, vendor, newCommentsLength})

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

// SplitByBlock splits the file into two parts exactly at the end of the block content
func (flac *Flac) splitByBlock(block *MetadataBlock) ([]byte, []byte, error) {

	blockIndex := block.Index

	// Read from file start until the start of the current block we want to remov
	flac.file.Seek(0, 0)
	// Get the current block (the one we want to remove) size
	currentBlockSize := int64(4 + block.BlockHeader.BlockLength)
	previousData, err := flac.readBytes(int(blockIndex) + int(currentBlockSize))

	if err != nil {
		return nil, nil, fmt.Errorf("unable to read bytes: %w", err)
	}

	// Move to the end of the current block size we want to remove and read the rest of the file
	flac.file.Seek(blockIndex+currentBlockSize, 0)
	postData, err := flac.readBytes(int(flac.fileSize) - (int(blockIndex) + int(currentBlockSize)))

	if err != nil {
		return nil, nil, fmt.Errorf("unable to split file: %w", err)
	}

	return previousData, postData, nil
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

// SetMetadata inserts a new metadata inside the FLAC file, if it doesn't exists it creates it otherwise it updates the value.
func (flac *Flac) SetMetadata(title string, value string) error {
	flac.pendingComments = append(flac.pendingComments, VorbisComment{
		Title: title,
		Value: value,
	})

	return nil
}

func (flac *Flac) BulkAddMetadata(meta FlacMetadatas) error {
	fields := map[string]string{
		"title":  meta.Title,
		"artist": meta.Artist,
		"album":  meta.Album,
		"date":   meta.Date,
	}

	for key, value := range fields {
		if value == "" {
			continue
		}
		if err := flac.SetMetadata(key, value); err != nil {
			return fmt.Errorf("unable to set %s in BulkAddMetadata: %w", key, err)
		}
	}

	if len(meta.Cover) > 0 {
		if err := flac.SetCoverPictureFromBytes(meta.Cover); err != nil {
			return fmt.Errorf("unable to set cover in BulkAddMetadata: %w", err)
		}
	}

	return nil
}

// RemoveMetadata from the currently opened flac file.
// If IgnoreIfMissing is set to true then no error will be returned if the
// metadata key is missing.
func (flac *Flac) RemoveMetadata(title string, ignoreIfMissing bool) error {
	exists := false
	if !ignoreIfMissing {
		for _, cmt := range flac.parsedComments {
			if strings.EqualFold(title, cmt.Title) {
				exists = true
				break
			}
		}
		if !exists {
			return fmt.Errorf("unable to remove metadata with name '%s' because it doesn't exist in file", title)
		}
	}

	updatedComments := make([]VorbisComment, 0)

	for _, cmt := range flac.pendingComments {
		if !strings.EqualFold(cmt.Title, title) {
			updatedComments = append(updatedComments, cmt)
		}
	}

	flac.pendingComments = updatedComments
	flac.removedComments[strings.ToLower(title)] = true

	return nil
}

// SetCoverPicture sets a cover picture for the current FLAC file, if already exists then it overwrites it
// Also add the ability to add image directly from buffer not necessarily from a given downloaded file
func (flac *Flac) SetCoverPictureFromPath(filePath string) error {
	pictureMimeType, pictureBytes, err := flac.parseImageFromPath(filePath)

	if err != nil {
		return fmt.Errorf("unable to parse image with path: '%s': %w", filePath, err)
	}

	pictureBlockBytes, err := flac.createPictureBlock(pictureBytes, *pictureMimeType)

	if err != nil {
		return fmt.Errorf("unable to add cover picture: %w", err)
	}

	flac.pendingCoverPicture = pictureBlockBytes

	return nil
}

func (flac *Flac) SetCoverPictureFromBytes(imgBytes []byte) error {
	if len(imgBytes) < 512 {
		return fmt.Errorf("unable to detect content type, image is too small or format is broken")
	}

	pictureMimeType := http.DetectContentType(imgBytes[:512])
	pictureBlockBytes, err := flac.createPictureBlock(imgBytes, pictureMimeType)

	if err != nil {
		return fmt.Errorf("unable to add cover picture: %w", err)
	}

	flac.pendingCoverPicture = pictureBlockBytes

	return nil
}

func (flac *Flac) RemoveCoverPicture(ignoreIfMissing bool) error {
	if flac.parsedCoverPicture == nil {
		if !ignoreIfMissing {
			return fmt.Errorf("unable to remove cover picture: opened flac file doesn't have one")
		}
	}

	flac.removeCoverPicture = true

	return nil
}

// getMetadataEndOffset returns the byte offset in the file where metadata ends
func (flac *Flac) getMetadataEndOffset() (int64, error) {
	offset := int64(4) // skip 'fLaC' header
	for {
		headerBytes := make([]byte, 4)
		if _, err := flac.file.ReadAt(headerBytes, offset); err != nil {
			return 0, fmt.Errorf("unable to read metadata header at offset %d: %w", offset, err)
		}

		isLast := (headerBytes[0] & 0x80) != 0
		blockLength := int64(binary.BigEndian.Uint32(append([]byte{0}, headerBytes[1:]...)))

		offset += 4 + blockLength

		if isLast {
			break
		}
	}

	return offset, nil
}

func (flac *Flac) Save(outputPath *string) error {
	var metadataBuffer []byte
	var rawAudioBuffer []byte

	// Read all metadata blocks
	blocks, err := flac.readAllMetadataBlocks()
	if err != nil {
		return fmt.Errorf("unable to read all metadata blocks: %w", err)
	}

	// Rebuilding the FLAC file
	// First thing first add the FLAC magic header 'fLaC'
	flac.file.Seek(0, 0)
	magicHeader, err := flac.readBytes(4)
	if err != nil {
		return fmt.Errorf("failed to read FLAC header: %w", err)
	}

	// Prepare new metadata blocks buffer
	newBlocks := []MetadataBlock{}

	// STREAMINFO block is mandatory
	streamInfo, err := flac.getBlock("STREAMINFO")
	if err != nil {
		return fmt.Errorf("missing STREAMINFO block: %w", err)
	}
	newBlocks = append(newBlocks, *streamInfo)

	// VORBIS_COMMENT
	if len(flac.pendingComments) > 0 {
		vorbisBlock, err := flac.createVorbisBlock()
		if err != nil {
			return fmt.Errorf("failed to create VORBIS_COMMENT: %w", err)
		}
		newBlocks = append(newBlocks, MetadataBlock{
			BlockHeader: MetadataBlockHeader{Data: vorbisBlock[:4]}, // placeholder header
			BlockData:   vorbisBlock[4:],
		})
	} else if flac.vorbisIndex != nil {
		vorbisBlock, _ := flac.getBlock("VORBIS_COMMENT")
		newBlocks = append(newBlocks, *vorbisBlock)
	}

	// Cover picture
	if len(flac.pendingCoverPicture) > 0 {
		newBlocks = append(newBlocks, MetadataBlock{
			BlockHeader: MetadataBlockHeader{Data: flac.pendingCoverPicture[:4]},
			BlockData:   flac.pendingCoverPicture[4:],
		})
	} else if flac.parsedCoverPicture != nil && !flac.removeCoverPicture {
		newBlocks = append(newBlocks, *flac.parsedCoverPicture)
	}

	// Other filtered blocks
	filteredBlocks := GetFilteredBlocks(blocks, []string{
		"STREAMINFO", "VORBIS_COMMENT", "PICTURE",
	})
	for _, b := range filteredBlocks {
		newBlocks = append(newBlocks, b)
	}

	// Mark the last block correctly
	for i := range newBlocks {
		header := newBlocks[i].BlockHeader.Data
		if i == len(newBlocks)-1 {
			// Set isLastBlock bit
			header[0] |= 0x80
		} else {
			// Clear isLastBlock bit
			header[0] &^= 0x80
		}
		newBlocks[i].BlockHeader.Data = header
		metadataBuffer = AppendTo(metadataBuffer, [][]byte{header, newBlocks[i].BlockData})
	}

	// Read raw audio starting after the original metadata
	metadataEnd, err := flac.getMetadataEndOffset()
	if err != nil {
		return fmt.Errorf("cannot get metadata end offset: %w", err)
	}
	flac.file.Seek(metadataEnd, 0)
	rawAudioBuffer, err = io.ReadAll(flac.file)
	if err != nil {
		return fmt.Errorf("unable to read raw audio: %w", err)
	}

	// Create output file
	outFileName := flac.fileName
	if outputPath != nil {
		outFileName = *outputPath
	}
	outFile, err := os.Create(outFileName)
	if err != nil {
		return fmt.Errorf("unable to create file '%s': %w", outFileName, err)
	}
	defer outFile.Close()

	// Write FLAC file: magic header + metadata + raw audio
	fullBuffer := AppendTo(nil, [][]byte{magicHeader, metadataBuffer, rawAudioBuffer})
	if _, err := outFile.Write(fullBuffer); err != nil {
		return fmt.Errorf("unable to write FLAC file: %w", err)
	}

	return nil
}
