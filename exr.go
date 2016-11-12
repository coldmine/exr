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

func Decode(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)

	// EXR file have little endian form.
	parse := binary.LittleEndian

	// Magic number: 4 bytes
	magicByte := make([]byte, 4)
	r.Read(magicByte)
	magic := int(parse.Uint32(magicByte))
	if magic != MagicNumber {
		return nil, fmt.Errorf("wrong magic number: %v, need %v", magic, MagicNumber)
	}

	// Version field: 4 bytes
	// first byte: version number
	// 2-4  bytes: set of boolean flags
	versionByte := make([]byte, 4)
	r.Read(versionByte)
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
	attrs := make([]attribute, 0)
	for {
		pAttr, err := parseAttribute(r, parse)
		if err != nil {
			fmt.Println("Could not read header: ", err)
			os.Exit(1)
		}
		if pAttr == nil {
			// Header ends.
			break
		}
		attr := *pAttr
		fmt.Println(attr.name, attr.size)
		attrs = append(attrs, attr)
	}

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
	if len(nameByte) == 1 {
		// Header ends.
		return nil, nil
	}
	// TODO: Properly validate length of attribute name.
	if len(nameByte) > 255 {
		return nil, errors.New("attribute name too long.")
	}
	name := string(nameByte)

	typeByte, err := r.ReadBytes(0x00)
	if err != nil {
		return nil, err
	}
	typ := string(typeByte)
	// TODO: Should I validate the length of attribute type?

	sizeByte := make([]byte, 4)
	_, err = r.Read(sizeByte)
	if err != nil {
		return nil, err
	}
	size := int(parse.Uint32(sizeByte))

	valueByte := make([]byte, 0, size)
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
		valueByte = append(valueByte, b...)
	}

	attr := attribute{
		name:  name,
		typ:   typ,
		size:  size,
		value: valueByte,
	}
	return &attr, nil
}
func fromScanLineFile() {}

func fromSinglePartFile() {}

func fromMultiPartFile() {}
