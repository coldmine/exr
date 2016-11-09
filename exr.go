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

	// TODO: parse boolean flags
	return nil, nil
}

func fromScanLineFile() {}

func fromSinglePartFile() {}

func fromMultiPartFile() {}
