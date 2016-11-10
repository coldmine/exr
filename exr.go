package exr

import (
	"bufio"
	"encoding/binary"
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

	// Magic number: 4 bytes
	magicByte := make([]byte, 4)
	r.Read(magicByte)
	magic := int(binary.LittleEndian.Uint32(magicByte))
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
	versionInt := int(binary.LittleEndian.Uint32(versionByte))
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

	return nil, nil
}

func fromScanLineFile() {}

func fromSinglePartFile() {}

func fromMultiPartFile() {}
