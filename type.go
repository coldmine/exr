package exr

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

type box2f struct {
	xMin float32
	yMin float32
	xMax float32
	yMax float32
}

type channel struct {
	name      string
	pixelType int32
	pLinear   uint8
	xSampling int32
	ySampling int32
}

type chlist []channel

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

type compression uint8

type envmap uint8

type keycode struct {
	filmMfcCode   int32
	filmType      int32
	prefix        int32
	count         int32
	perfOffset    int32
	perfsPerFrame int32
	perfsPerCount int32
}

type lineOrder uint8

type m33f [9]float32

type m44f [16]float32

type preview struct {
	width  int32
	height int32
	data   []byte
}

type rational struct {
	a int32
	b uint32
}

type tiledesc struct {
	xSize uint32
	ySize uint32
	mode  uint8
}

type timecode struct {
	timeAndFlags uint32
	userData     uint32
}

type v2i [2]int32

type v2f [2]float32

type v3i [3]int32

type v3f [3]float32
