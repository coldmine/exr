package exr

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"log"
	"math"
	"os"
)

// A FormatError reports that the input is not a valid EXR image.
type FormatError string

func (e FormatError) Error() string {
	return "exr: invalid format: " + string(e)
}

// An UnsupportedError reports that the input uses a valid but
// unimplemented feature.
type UnsupportedError string

func (e UnsupportedError) Error() string {
	return "exr: unsupported feature: " + string(e)
}

var MagicNumber = 20000630

// EXR file have little endian form.
var parse = binary.LittleEndian

var numLinesPerBlock = map[compression]int{
	NO_COMPRESSION:    1,
	RLE_COMPRESSION:   1,
	ZIPS_COMPRESSION:  1,
	ZIP_COMPRESSION:   16,
	PIZ_COMPRESSION:   32,
	PXR24_COMPRESSION: 16,
	B44_COMPRESSION:   32,
	B44A_COMPRESSION:  32,
}

type VersionField struct {
	// version is version of an exr image.
	version int

	// tiled indicates the image is tiled or scanline image.
	// This value is valid only if the image is single part.
	// (multiPart == false)
	tiled bool

	// longName indicates the image could have long(maximum: 255 bytes) attribute or channel names.
	// When it is false, image could have only short(maximum: 31 bytes) names.
	longName bool

	// deep indicates the image is deep image or plane image.
	deep bool

	// multiPart indicates the image is consists of multi parts or a single part.
	multiPart bool
}

func Decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)

	// Magic number: 4 bytes
	magicByte, err := read(r, 4)
	if err != nil {
		return nil, err
	}
	magic := int(parse.Uint32(magicByte))
	if magic != MagicNumber {
		return nil, FormatError(fmt.Sprintf("wrong magic number"))
	}

	// version field: 4 bytes
	// first byte: version number
	// next 3 bytes: set of boolean flags
	versionBytes, err := read(r, 4)
	if err != nil {
		return nil, err
	}
	versionNum := int(parse.Uint32(versionBytes))

	vf := VersionField{
		version:   int(versionBytes[0]),
		tiled:     versionNum&0x200 != 0,
		longName:  versionNum&0x400 != 0,
		deep:      versionNum&0x800 != 0,
		multiPart: versionNum&0x1000 != 0,
	}
	if vf.tiled {
		if vf.deep {
			return nil, FormatError("single tile bit is on, non-image bit should be off")
		}
		if vf.multiPart {
			return nil, FormatError("single tile bit is on, multi-part bit should be off")
		}
	}

	fmt.Println("version: ", vf.version)
	fmt.Println("tiled: ", vf.tiled)
	fmt.Println("multi part: ", vf.multiPart)
	fmt.Println("deep: ", vf.deep)
	fmt.Println("long name: ", vf.longName)

	// Parse attributes of a header.
	parts := make([]map[string]attribute, 0)

	for {
		fmt.Println("== a part ==")

		header := make(map[string]attribute)
		for {
			pAttr, err := parseAttribute(r, parse)
			if err != nil {
				return nil, err
			}
			if pAttr == nil {
				// Single header ends.
				break
			}
			attr := *pAttr
			header[attr.name] = attr
		}
		parts = append(parts, header)

		if !vf.multiPart {
			break
		}
		bs, err := r.Peek(1)
		if err != nil {
			return nil, err
		}
		if bs[0] == 0x00 {
			break
		}
	}

	// TODO: Parse multi-part image.
	if vf.multiPart {
		log.Fatal("does not support multi-part image yet")
	}
	header := parts[0]

	for _, attr := range header {
		switch attr.typ {
		case "float":
			fmt.Println(attr.name, math.Float32frombits(parse.Uint32(attr.value)))
		case "box2i":
			fmt.Println(attr.name, box2iFromBytes(attr.value))
		case "box2f":
			fmt.Println(attr.name, box2fFromBytes(attr.value))
		case "chlist":
			fmt.Println(attr.name, chlistFromBytes(attr.value))
		case "chromaticities":
			fmt.Println(attr.name, chromaticitiesFromBytes(attr.value))
		case "compression":
			fmt.Println(attr.name, compressionFromBytes(attr.value))
		case "envmap":
			fmt.Println(attr.name, envmapFromBytes(attr.value))
		case "keycode":
			fmt.Println(attr.name, keycodeFromBytes(attr.value))
		case "lineOrder":
			fmt.Println(attr.name, lineOrderFromBytes(attr.value))
		case "m33f":
			fmt.Println(attr.name, m33fFromBytes(attr.value))
		case "m44f":
			fmt.Println(attr.name, m44fFromBytes(attr.value))
		case "preview":
			// long result
			// fmt.Println(attr.name, previewFromBytes(attr.value))
		case "rational":
			fmt.Println(attr.name, rationalFromBytes(attr.value))
		case "string":
			fmt.Println(attr.name, string(attr.value))
		case "tiledesc":
			fmt.Println(attr.name, tiledescFromBytes(attr.value))
		case "timecode":
			fmt.Println(attr.name, timecodeFromBytes(attr.value))
		case "v2i":
			fmt.Println(attr.name, v2iFromBytes(attr.value))
		case "v2f":
			fmt.Println(attr.name, v2fFromBytes(attr.value))
		case "v3i":
			fmt.Println(attr.name, v3iFromBytes(attr.value))
		case "v3f":
			fmt.Println(attr.name, v3fFromBytes(attr.value))
		default:
			fmt.Printf("unknown type of attribute %q: %v\n", attr.name, attr.typ)
		}
	}

	// Parse channels.
	channelsAttr, ok := header["channels"]
	if !ok {
		return nil, FormatError("header does not have 'channels' attribute")
	}
	channels := chlistFromBytes(channelsAttr.value)
	fmt.Println("channels: ", channels)

	// Check image (x, y) size.
	dataWindowAttr, ok := header["dataWindow"]
	if !ok {
		return nil, FormatError("header does not have 'dataWindow' attribute")
	}
	dataWindow := box2iFromBytes(dataWindowAttr.value)
	fmt.Println("data window: ", dataWindow)

	// Check compression method.
	compressionAttr, ok := header["compression"]
	if !ok {
		return nil, FormatError("header does not have 'compression' attribute")
	}
	compressionMethod := compressionFromBytes(compressionAttr.value)
	blockLines := numLinesPerBlock[compressionMethod]
	fmt.Printf("compression method: %v - %v lines per block\n", compressionMethod, blockLines)

	lineOrderAttr, ok := header["lineOrder"]
	if !ok {
		return nil, FormatError("header does not have 'lineOrder' attribute")
	}
	lineOrder := lineOrderFromBytes(lineOrderAttr.value)
	fmt.Printf("line order: %v\n", lineOrder)

	// Parse offsets.
	nLines := int(dataWindow.yMax - dataWindow.yMin + 1)
	nChunks := nLines / blockLines
	if nLines%blockLines != 0 {
		nChunks++
	}
	fmt.Printf("number of chunks: %d = %d/%d\n", nChunks, nLines, blockLines)
	offsets := make([]uint64, nChunks)
	for i := range offsets {
		offsetByte, err := read(r, 8)
		if err != nil {
			return nil, err
		}
		offsets[i] = uint64(parse.Uint64(offsetByte))
	}
	fmt.Println(offsets)

	for _, o := range offsets {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		r := bufio.NewReader(f)
		_, err = r.Discard(int(o))
		if err != nil {
			log.Fatal("discard failed")
		}
		n, err := read(r, 4)
		if err != nil {
			log.Fatal("read offset failed")
		}
		y := parse.Uint32(n)
		n, err = read(r, 4)
		if err != nil {
			log.Fatal("read offset failed")
		}
		size := int(parse.Uint32(n))
		compressed, err := read(r, size)
		if err != nil {
			log.Fatal("read compressed data failed")
		}
		width := int(dataWindow.xMax - dataWindow.xMin + 1)
		b := newBlockInfo(compressionMethod, channels, width)
		raw := decompress(compressed, b)
		fmt.Println(y, size, channels, len(raw))
	}
	return nil, nil
}

func decompress(compressed []byte, block blockInfo) []uint16 {
	if block.compression == PIZ_COMPRESSION {
		return nil
		// return pizDecompress(compressed, block)
	}
	log.Fatalf("decompress of %d not supported yet", block.compression)
	return nil
}

// blockInfo contains information of block to compress or decompress the images.
type blockInfo struct {
	compression compression
	channels    chlist
	width       int
	height      int
}

func newBlockInfo(c compression, channels chlist, width int) blockInfo {
	return blockInfo{
		compression: c,
		channels:    channels,
		width:       width,
		height:      numLinesPerBlock[c],
	}
}

type attribute struct {
	name  string
	typ   string
	size  int
	value []byte // TODO: parse it.
}

// parseAttribute parses an attribute of a header.
//
// It returns one of following forms.
//
// 	(*attribute, nil) if it reads from reader well.
// 	(nil, error) if any error occurred when read.
// 	(nil, nil) if the header ends.
//
func parseAttribute(r *bufio.Reader, parse binary.ByteOrder) (*attribute, error) {
	nameByte, err := r.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}
	nameByte = nameByte[:len(nameByte)-1] // remove trailing 0x00
	if len(nameByte) == 0 {
		// Header ends.
		return nil, nil
	}
	// TODO: Properly validate length of attribute name.
	if len(nameByte) > 255 {
		return nil, FormatError("attribute name too long.")
	}
	name := string(nameByte)

	typeByte, err := r.ReadBytes(0x00)
	typeByte = typeByte[:len(typeByte)-1] // remove trailing 0x00
	if err != nil {
		return nil, err
	}
	typ := string(typeByte)
	// TODO: Should I validate the length of attribute type?

	sizeByte, err := read(r, 4)
	if err != nil {
		return nil, err
	}
	size := int(parse.Uint32(sizeByte))

	valueByte, err := read(r, size)
	if err != nil {
		return nil, err
	}

	attr := attribute{
		name:  name,
		typ:   typ,
		size:  size,
		value: valueByte,
	}
	return &attr, nil
}

// read reads _size_ bytes from *bufio.Reader and return it as ([]byte, error) form.
// If error occurs during read, it will return nil, error.
func read(r *bufio.Reader, size int) ([]byte, error) {
	bs := make([]byte, 0, size)
	remain := size
	for remain > 0 {
		s := remain
		if remain > bufio.MaxScanTokenSize {
			s = bufio.MaxScanTokenSize
		}
		b := make([]byte, s)
		n, err := r.Read(b)
		if err != nil {
			return nil, err
		}
		b = b[:n]
		remain -= n
		bs = append(bs, b...)
	}
	return bs, nil
}
