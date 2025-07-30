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

type MetadataBlock struct {
	Index       int64
	BlockType   string
	IsLastBlock bool
	BlockData   []byte
}

type VorbisComment struct {
	Title string
	Value string
}

type Flac struct {
	file *os.File
}

func Open(path string) (*Flac, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Failed to initialize flacgo %w", err)
	}

	return &Flac{
		file: f,
	}, nil
}

func (flac *Flac) GetAsText(array []byte) string {
	var outputString string
	for _, integer := range array {
		outputString += string(integer)
	}
	return outputString
}

// Better handle possible errors and update everywhere this function is called to better handle the error scenario
func (flac *Flac) ReadBytes(bytesNum int) []byte {
	data := make([]byte, bytesNum)
	flac.file.Read(data)
	return data
}

func (flac *Flac) ReadMetadataBlock(offset int64) MetadataBlock {
	flac.file.Seek(offset, io.SeekStart)
	blockHeader := flac.ReadBytes(1)

	blockType := blockHeader[0] & 0x7F
	isLastBlock := (blockHeader[0] & 0x80) >> 7

	lengthBytes := flac.ReadBytes(3)

	// Uint32 requires 4 bytes slice to convert to Uint32 so I add one before as 0x00
	blockLength := binary.BigEndian.Uint32(append([]byte{0}, lengthBytes...))

	blockContent := flac.ReadBytes(int(blockLength))

	return MetadataBlock{
		Index:       offset,
		BlockType:   BlockMapping[blockType],
		IsLastBlock: isLastBlock == 1,
		BlockData:   blockContent,
	}
}

func (flac *Flac) ReadAllMetadataBlocks() []MetadataBlock {
	var offset int = 4
	blocks := []MetadataBlock{}

	for true {
		data := flac.ReadMetadataBlock(int64(offset))
		offset += 4 + len(data.BlockData)
		blocks = append(blocks, data)

		if data.IsLastBlock {
			break
		}
	}

	return blocks
}

func (flac *Flac) toBytes(value uint32, bytesLength int, endian binary.ByteOrder) []byte {
	tmpBuffer := make([]byte, 4)
	endian.PutUint32(tmpBuffer, value)
	return tmpBuffer[4-bytesLength:]
}

// Binary data should contain a vorbis payload as bytes
// ex: b'\r\x00\x00\x00Lavf59.27.100\x07\x00\x00\x00!\x00\x00\x00title=London Calling (Remastered)\x10\x00\x00\x00artist=The Clash\x15\x00\x00\x00ALBUMARTIST=The Clash\x15\x00\x00\x00album=London Calling \x0f\x00\x00\x00date=1979-12-14\x15\x00\x00\x00genre=Punk / New Wave\x15\x00\x00\x00encoder=Lavf59.27.100'
func (flac *Flac) ParseVorbisBlock(vorbisBlock []byte) []VorbisComment {
	var vorbisComments []VorbisComment

	vendorLength := binary.LittleEndian.Uint32(vorbisBlock[0:4])
	numberOfComments := binary.LittleEndian.Uint32(vorbisBlock[4+vendorLength : 4+4+vendorLength])

	iteration := 0
	offset := 4 + 4 + vendorLength
	for iteration < int(numberOfComments) {
		commentLength := binary.LittleEndian.Uint32(vorbisBlock[offset : offset+4])
		commentContent := string(vorbisBlock[offset+4 : offset+4+commentLength])

		values := strings.Split(commentContent, "=")

		vorbisComments = append(vorbisComments, VorbisComment{
			Title: values[0],
			Value: values[1],
		})

		offset += commentLength + 4
		iteration += 1
	}

	return vorbisComments
}

/*
Use when metadata is missing otherwise use already existing one?
By specification there must be only one VORBIS_COMMENT at a time in a flac file
Fields example: {"title": "London Calling", ...}
*/
func (flac *Flac) CreateVorbisBlock(fields map[string]string) error {
	blockType := 4 // 4 = VORBIS_COMMENT
	var lastBlockIndex int64
	allBlocks := flac.ReadAllMetadataBlocks()

	for _, block := range allBlocks {
		if block.BlockType == "VORBIS_COMMENT" {
			// Can't add a new VORBIS_COMMENT then
			return fmt.Errorf("You can have only one VORBIS_COMMENT per .flac file and there's one alredy.\n")
		}
		if block.IsLastBlock {
			lastBlockIndex = block.Index
		}
	}

	var body []byte

	vendor := []byte("flacgo1.1")
	vendorLength := flac.toBytes(uint32(len(vendor)), 4, binary.LittleEndian)
	newCommentsLength := flac.toBytes(uint32(len(fields)), 4, binary.LittleEndian)

	body = append(body, vendorLength...)
	body = append(body, vendor...)
	body = append(body, newCommentsLength...)

	for key, value := range fields {
		comment := []byte(fmt.Sprintf("%s=%s", key, value))
		commentLength := flac.toBytes(uint32(len(comment)), 4, binary.LittleEndian)

		body = append(body, commentLength...)
		body = append(body, comment...)
	}

	isLast := 0
	payloadLength := flac.toBytes(uint32(len(body)), 3, binary.BigEndian)
	headerByte := (isLast << 7) | blockType

	header := []byte{byte(headerByte)}
	header = append(header, payloadLength...)

	fileInfo, err := flac.file.Stat()
	if err != nil {
		return fmt.Errorf("Failed to stat file: %w", err)
	}
	totalSize := fileInfo.Size()

	// prevData represnts file bytes from start to last block index
	flac.file.Seek(0, 0)
	prevData := flac.ReadBytes(int(lastBlockIndex))

	// postData represnts file bytes from last block index to file end
	flac.file.Seek(lastBlockIndex, 0)
	postData := flac.ReadBytes(int(totalSize - lastBlockIndex))

	newBuffer := append(prevData, header...)
	newBuffer = append(newBuffer, body...)
	newBuffer = append(newBuffer, postData...)

	// TODO: make output name arbitrary from the user
	err = os.WriteFile("output.flac", newBuffer, 0644)

	if err != nil {
		return fmt.Errorf("Failed to write file: %w", err)
	}

	return nil
}
