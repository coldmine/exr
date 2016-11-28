package exr

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"os"
)

var MagicNumber = 20000630

type compressionType int

const (
	NoCompression = compressionType(iota)
	RLECompression
	ZIPSCompression
	ZIPCompression
	PIZCompression
	PXR24Compression
	B44Compression
	B44ACompression
)

var numLinesPerBlock = map[compressionType]int{
	NoCompression:    1,
	RLECompression:   1,
	ZIPSCompression:  1,
	ZIPCompression:   16,
	PIZCompression:   32,
	PXR24Compression: 16,
	B44Compression:   32,
	B44ACompression:  32,
}

func Decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)

	// EXR file have little endian form.
	parse := binary.LittleEndian

	// Magic number: 4 bytes
	magicByte, err := read(r, 4)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	magic := int(parse.Uint32(magicByte))
	if magic != MagicNumber {
		return nil, fmt.Errorf("wrong magic number: %v, need %v", magic, MagicNumber)
	}

	// Version field: 4 bytes
	// first byte: version number
	// 2-4  bytes: set of boolean flags
	versionByte, err := read(r, 4)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	version := int(versionByte[0])
	fmt.Println(version)

	// Parse image type
	var singlePartScanLine bool
	var singlePartTiled bool
	var singlePartDeep bool
	var multiPart bool
	var multiPartDeep bool
	versionInt := int(parse.Uint32(versionByte))
	if versionInt&0x200 != 0 {
		singlePartTiled = true
	}
	if !singlePartTiled {
		deep := false
		if versionInt&0x800 != 0 {
			deep = true
		}
		multi := false
		if versionInt&0x1000 != 0 {
			multi = true
		}
		if multi && !deep {
			multiPart = true
		} else if multi && deep {
			multiPartDeep = true
		} else if !multi && deep {
			singlePartDeep = true
		} else {
			singlePartScanLine = true
		}
	}
	if singlePartScanLine {
		fmt.Println("It is single-part scanline image.")
	} else if singlePartTiled {
		fmt.Println("It is single-part tiled image.")
	} else if singlePartDeep {
		fmt.Println("It is single-part deep image.")
	} else if multiPart {
		fmt.Println("It is multi-part image.")
	} else if multiPartDeep {
		fmt.Println("It is multi-part deep image.")
	}

	// Check image could have long attribute name
	var longAttrName bool
	if versionInt&0x400 != 0 {
		longAttrName = true
	}
	if longAttrName {
		fmt.Println("It could have long attribute names")
	}

	// Parse attributes of a header.
	parts := make([]map[string]attribute, 0)

	for i := 0; ; i++ {
		fmt.Println("== a part ==")

		header := make(map[string]attribute)
		for {
			pAttr, err := parseAttribute(r, parse)
			if err != nil {
				fmt.Println("Could not read header: ", err)
				os.Exit(1)
			}
			if pAttr == nil {
				// Single header ends.
				break
			}
			attr := *pAttr
			fmt.Println(attr.name, attr.size)
			header[attr.name] = attr
		}
		parts = append(parts, header)

		if !multiPart && !multiPartDeep {
			break
		}
		bs, err := r.Peek(1)
		if err != nil {
			fmt.Println("Could not peek:", err)
			os.Exit(1)
		}
		if bs[0] == 0x00 {
			break
		}
	}

	// TODO: Parse multi-part image.
	header := parts[0]

	// Check image (x, y) size.
	dataWindow, ok := header["dataWindow"]
	if !ok {
		fmt.Println("Header does not have 'dataWindow' attribute")
		os.Exit(1)
	}
	var xMin, yMin, xMax, yMax int
	xMin = int(parse.Uint32(dataWindow.value[0:4]))
	yMin = int(parse.Uint32(dataWindow.value[4:8]))
	xMax = int(parse.Uint32(dataWindow.value[8:12]))
	yMax = int(parse.Uint32(dataWindow.value[12:16]))
	fmt.Println(xMin, yMin, xMax, yMax)

	// Check compression method.
	compression, ok := header["compression"]
	if !ok {
		fmt.Println("Header does not have 'compression' attribute")
		os.Exit(1)
	}
	compressionMethod := compressionType(uint8(compression.value[0]))
	blockLines := numLinesPerBlock[compressionMethod]
	fmt.Println(blockLines)

	// Parse offsets.
	offsets := make([]uint64, 0)
	for i := yMin; i <= yMax; i += blockLines {
		offsetByte, err := read(r, 8)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		offset := uint64(parse.Uint64(offsetByte))
		offsets = append(offsets, offset)
	}
	fmt.Println(offsets)

	return nil, nil
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
		return nil, errors.New("attribute name too long.")
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
