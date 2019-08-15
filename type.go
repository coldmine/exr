package exr

import (
	"bufio"
	"bytes"
	"log"
)

// These types are defined by IlmImf.
// Please see following document.
//
// 	OpenEXRFileLayout.pdf - Predefined Attribute Types
//

type box2i struct {
	xMin int32
	yMin int32
	xMax int32
	yMax int32
}

func box2iFromBytes(b []byte) box2i {
	if len(b) != 16 {
		log.Fatal("box2iFromBytes: need bytes of length 16")
	}
	return box2i{
		xMin: int32(parse.Uint32(b[0:4])),
		yMin: int32(parse.Uint32(b[4:8])),
		xMax: int32(parse.Uint32(b[8:12])),
		yMax: int32(parse.Uint32(b[12:16])),
	}
}

type box2f struct {
	xMin float32
	yMin float32
	xMax float32
	yMax float32
}

func box2fFromBytes(b []byte) box2f {
	if len(b) != 16 {
		log.Fatal("box2fFromBytes: need bytes of length 16")
	}
	return box2f{
		xMin: float32(parse.Uint32(b[0:4])),
		yMin: float32(parse.Uint32(b[4:8])),
		xMax: float32(parse.Uint32(b[8:12])),
		yMax: float32(parse.Uint32(b[12:16])),
	}
}

type channel struct {
	name      string
	pixelType int32
	pLinear   uint8
	xSampling int32
	ySampling int32
}

type chlist []channel

func chlistFromBytes(b []byte) chlist {
	chans := make(chlist, 0)
	buf := bufio.NewReader(bytes.NewBuffer(b))
	for {
		nameByte, err := buf.ReadBytes(0x00)
		if err != nil {
			log.Fatal(err)
		}
		name := string(nameByte[:len(nameByte)-1])

		channelBytes, err := read(buf, 16)
		if err != nil {
			log.Fatal(err)
		}
		pixelType := int32(parse.Uint32(channelBytes[:4]))
		pLinear := uint8(channelBytes[4])
		// channelBytes[5:8] are place holders.
		xSampling := int32(parse.Uint32(channelBytes[8:12]))
		ySampling := int32(parse.Uint32(channelBytes[12:]))
		ch := channel{
			name:      name,
			pixelType: pixelType,
			pLinear:   pLinear,
			xSampling: xSampling,
			ySampling: ySampling,
		}
		chans = append(chans, ch)
		if buf.Buffered() == 1 {
			nullByte, err := buf.Peek(1)
			if err != nil {
				log.Fatal(err)
			}
			if nullByte[0] != 0x00 {
				log.Fatal(FormatError("channels are must seperated by a null byte"))
			}
			break
		}
	}
	return chans
}

type chromatics struct {
	redX   float32
	redY   float32
	greenX float32
	greenY float32
	blueX  float32
	blueY  float32
	whiteX float32
	whiteY float32
}

func chromaticsFromBytes(b []byte) chromatics {
	if len(b) != 32 {
		log.Fatal("chromaticsFromBytes: need bytes of length 32")
	}
	return chromatics{
		redX:   float32(parse.Uint32(b[0:4])),
		redY:   float32(parse.Uint32(b[4:8])),
		greenX: float32(parse.Uint32(b[8:12])),
		greenY: float32(parse.Uint32(b[12:16])),
		blueX:  float32(parse.Uint32(b[16:20])),
		blueY:  float32(parse.Uint32(b[20:24])),
		whiteX: float32(parse.Uint32(b[24:28])),
		whiteY: float32(parse.Uint32(b[28:32])),
	}
}

type compression uint8

const (
	NO_COMPRESSION = compression(iota)
	RLE_COMPRESSION
	ZIPS_COMPRESSION
	ZIP_COMPRESSION
	PIZ_COMPRESSION
	PXR24_COMPRESSION
	B44_COMPRESSION
	B44A_COMPRESSION
)

func (t compression) String() string {
	switch t {
	case NO_COMPRESSION:
		return "NO_COMPRESSION"
	case RLE_COMPRESSION:
		return "RLE_COMPRESSION"
	case ZIPS_COMPRESSION:
		return "ZIPS_COMPRESSION"
	case ZIP_COMPRESSION:
		return "ZIP_COMPRESSION"
	case PIZ_COMPRESSION:
		return "PIZ_COMPRESSION"
	case PXR24_COMPRESSION:
		return "PXR24_COMPRESSION"
	case B44_COMPRESSION:
		return "B44_COMPRESSION"
	case B44A_COMPRESSION:
		return "B44A_COMPRESSION"
	default:
		return "UNKNOWN_COMPRESSION"
	}
}

func compressionFromBytes(b []byte) compression {
	if len(b) != 1 {
		log.Fatal("compressionFromBytes: need bytes of length 1")
	}
	return compression(b[0])
}

type envmap uint8

func envmapFromBytes(b []byte) envmap {
	if len(b) != 1 {
		log.Fatal("envmapFromBytes: need bytes of length 1")
	}
	return envmap(b[0])
}

type keycode struct {
	filmMfcCode   int32
	filmType      int32
	prefix        int32
	count         int32
	perfOffset    int32
	perfsPerFrame int32
	perfsPerCount int32
}

func keycodeFromBytes(b []byte) keycode {
	if len(b) != 28 {
		log.Fatal("keycodeFromBytes: need bytes of length 28")
	}
	return keycode{
		filmMfcCode:   int32(parse.Uint32(b[:4])),
		filmType:      int32(parse.Uint32(b[4:8])),
		prefix:        int32(parse.Uint32(b[8:12])),
		count:         int32(parse.Uint32(b[12:16])),
		perfOffset:    int32(parse.Uint32(b[16:20])),
		perfsPerFrame: int32(parse.Uint32(b[20:24])),
		perfsPerCount: int32(parse.Uint32(b[24:28])),
	}
}

type lineOrder uint8

const (
	INCREASING_Y = lineOrder(iota)
	DECREASING_Y
	RANDOM_Y
)

func (l lineOrder) String() string {
	switch l {
	case INCREASING_Y:
		return "INCREASING_Y"
	case DECREASING_Y:
		return "DECREASING_Y"
	case RANDOM_Y:
		return "RANDOM_Y"
	default:
		return "UNKNOWN_LINE_ORDER"
	}
}

func lineOrderFromBytes(b []byte) lineOrder {
	if len(b) != 1 {
		log.Fatal("lineOrderFromBytes: need bytes of length 1")
	}
	return lineOrder(b[0])
}

type m33f [9]float32

func m33fFromBytes(b []byte) m33f {
	if len(b) != 36 {
		log.Fatal("m33fFromBytes: need bytes of length 36")
	}
	return [9]float32{
		float32(parse.Uint32(b[:4])),
		float32(parse.Uint32(b[4:8])),
		float32(parse.Uint32(b[8:12])),
		float32(parse.Uint32(b[12:16])),
		float32(parse.Uint32(b[16:20])),
		float32(parse.Uint32(b[20:24])),
		float32(parse.Uint32(b[24:28])),
		float32(parse.Uint32(b[28:32])),
		float32(parse.Uint32(b[32:36])),
	}
}

type m44f [16]float32

func m44fFromBytes(b []byte) m44f {
	if len(b) != 64 {
		log.Fatal("m44fFromBytes: need bytes of length 64")
	}
	return [16]float32{
		float32(parse.Uint32(b[:4])),
		float32(parse.Uint32(b[4:8])),
		float32(parse.Uint32(b[8:12])),
		float32(parse.Uint32(b[12:16])),
		float32(parse.Uint32(b[16:20])),
		float32(parse.Uint32(b[20:24])),
		float32(parse.Uint32(b[24:28])),
		float32(parse.Uint32(b[28:32])),
		float32(parse.Uint32(b[32:36])),
		float32(parse.Uint32(b[36:40])),
		float32(parse.Uint32(b[40:44])),
		float32(parse.Uint32(b[44:48])),
		float32(parse.Uint32(b[48:52])),
		float32(parse.Uint32(b[52:56])),
		float32(parse.Uint32(b[56:60])),
		float32(parse.Uint32(b[60:64])),
	}
}

type preview struct {
	width  int32
	height int32
	data   []byte
}

func previewFromBytes(b []byte) preview {
	return preview{
		width:  int32(parse.Uint32(b[:4])),
		height: int32(parse.Uint32(b[4:8])),
		data:   b[8:],
	}
}

type rational struct {
	a int32
	b uint32
}

func rationalFromBytes(b []byte) rational {
	if len(b) != 8 {
		log.Fatal("rationalFromBytes: need bytes of length 8")
	}
	return rational{
		a: int32(parse.Uint32(b[:4])),
		b: parse.Uint32(b[4:8]),
	}
}

type tiledesc struct {
	xSize uint32
	ySize uint32
	mode  uint8
}

func tiledescFromBytes(b []byte) tiledesc {
	if len(b) != 9 {
		log.Fatal("tiledescFromBytes: need bytes of length 9")
	}
	return tiledesc{
		xSize: parse.Uint32(b[:4]),
		ySize: parse.Uint32(b[4:8]),
		mode:  b[8],
	}
}

type timecode struct {
	timeAndFlags uint32
	userData     uint32
}

func timecodeFromBytes(b []byte) timecode {
	if len(b) != 8 {
		log.Fatal("timecodeFromBytes: need bytes of length 8")
	}
	return timecode{
		timeAndFlags: parse.Uint32(b[:4]),
		userData:     parse.Uint32(b[4:8]),
	}
}

type v2i [2]int32

func v2iFromBytes(b []byte) v2i {
	if len(b) != 8 {
		log.Fatal("v2iFromBytes: need bytes of length 8")
	}
	return v2i{
		int32(parse.Uint32(b[:4])),
		int32(parse.Uint32(b[4:8])),
	}
}

type v2f [2]float32

func v2fFromBytes(b []byte) v2f {
	if len(b) != 8 {
		log.Fatal("v2fFromBytes: need bytes of length 8")
	}
	return v2f{
		float32(parse.Uint32(b[:4])),
		float32(parse.Uint32(b[4:8])),
	}
}

type v3i [3]int32

func v3iFromBytes(b []byte) v3i {
	if len(b) != 12 {
		log.Fatal("v3iFromBytes: need bytes of length 12")
	}
	return v3i{
		int32(parse.Uint32(b[:4])),
		int32(parse.Uint32(b[4:8])),
		int32(parse.Uint32(b[8:12])),
	}
}

type v3f [3]float32

func v3fFromBytes(b []byte) v3f {
	if len(b) != 12 {
		log.Fatal("v3fFromBytes: need bytes of length 12")
	}
	return v3f{
		float32(parse.Uint32(b[:4])),
		float32(parse.Uint32(b[4:8])),
		float32(parse.Uint32(b[8:12])),
	}
}
