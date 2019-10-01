package exr

import (
	"encoding/binary"
)

// piz compressed data structure
//
// [
// 	minimum non-zero byte index of bitmap
// 	maximum non-zero byte index of bitmap
// 	bitmap
// 	length of compressed data
// 	compressed data
// ]

func pizCompress(block blockInfo, raw []byte) []byte {
	if len(raw)%2 != 0 {
		panic("raw should []byte of size 2*n")
	}
	r := newByteReader(binary.LittleEndian, raw)

	// build bitmap from raw data
	bitm := newBitmap(1 << 16)
	for i := 0; i < len(raw); i += 2 {
		d := r.Uint16()
		bitm.Set(d)
	}
	bitm.Unset(0) // don't include zero in bitmap

	// apply forward lut to raw data
	r = newByteReader(binary.LittleEndian, raw)
	w := newByteWriter(binary.LittleEndian, raw)
	lut := forwardLutFromBitmap(bitm)
	for i := 0; i < len(raw); i += 2 {
		d := r.Uint16()
		w.Uint16(lut[d])
	}

	// wavlet encoding per channel
	var n, m int
	for _, ch := range block.channels {
		m += block.width * block.height * pixelSize(ch.pixelType)
		_ = n // avoid n declared and not used error, temporarily
		// TODO: applyWaveletEncode(raw[n:m], maxValue)
		n = m
	}

	// write
	compressed := make([]byte, len(raw)+12)
	w = newByteWriter(binary.LittleEndian, compressed)

	minNonZero := bitm.MinByteIndex()
	maxNonZero := bitm.MaxByteIndex()
	w.Uint32(uint32(minNonZero))
	w.Uint32(uint32(maxNonZero))
	w.Bytes(bitm[minNonZero : maxNonZero+1])

	cdata := huffmanCompress(block, raw)
	w.Uint32(uint32(len(cdata)))
	for _, d := range cdata {
		w.Uint8(d)
	}
	return compressed
}

func pizDecompress(block blockInfo, compressed []byte) []byte {
	r := newByteReader(binary.LittleEndian, compressed)

	// get bitmap info
	minNonZero := int(r.Uint16())
	maxNonZero := int(r.Uint16())
	bitm := newBitmap(1 << 16)
	copy(bitm[minNonZero:maxNonZero+1], r.Bytes(maxNonZero-minNonZero+1))

	// decompress
	lc := int(r.Uint32())
	cdata := r.Bytes(lc)
	raw := huffmanDecompress(block, cdata)

	// wavlet decode each channel
	var n, m int
	for _, ch := range block.channels {
		m += block.width * block.height * pixelSize(ch.pixelType)
		_ = n // avoid n declared and not used error, temporarily
		// TODO: applyWaveletDecode(raw[n:m], maxValue)
		n = m
	}

	// apply reverse lut
	r = newByteReader(binary.LittleEndian, raw)
	w := newByteWriter(binary.LittleEndian, raw)
	rlut := reverseLutFromBitmap(bitm)
	for i := 0; i < len(raw); i += 2 {
		d := r.Uint16()
		w.Uint16(rlut[d])
	}
	return raw
}
