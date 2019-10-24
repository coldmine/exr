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
		pixsize := pixelSize(ch.pixelType)
		m += block.width * block.height * pixsize
		// wav2Decode(raw[n:m], block.width, block.pixsize, block.height, block.width*pixsize, maxNonZero)
		n = m
	}
	_ = n

	// apply reverse lut
	lut := reverseLutFromBitmap(bitm)
	r = newByteReader(binary.LittleEndian, raw)
	w := newByteWriter(binary.LittleEndian, raw)
	for i := 0; i < len(raw); i += 2 {
		d := r.Uint16()
		w.Uint16(lut[d])
	}
	return raw
}

func wav2Decode(data []byte, nx, ox, ny, oy int, max int) {
	w14 := max < (1 << 14)

	// n is shorter side's length among width and height
	n := nx
	if n > ny {
		n = ny
	}
	// find a maximum number that is power of 2 smaller than n
	m2 := 1
	for m2 <= n {
		m2 <<= 1
	}
	m2 >>= 1
	m1 := m2 >> 1
	for m1 >= 1 {
		ox1 := m1 * ox
		ox2 := m2 * ox
		oy1 := m1 * oy
		oy2 := m2 * oy
		endy := ny * oy
		var d00, d01, d10, d11 uint16
		for iy := 0; iy <= endy-oy2; iy += oy2 {
			endx := iy + oy
			for ix := iy; ix <= endx-ox2; ix += ox2 {
				i00 := ix
				i01 := ix + ox1
				i10 := ix + oy1
				i11 := ix + ox1 + oy1
				d00 = getUint16(data[i00:])
				d01 = getUint16(data[i01:])
				d10 = getUint16(data[i10:])
				d11 = getUint16(data[i11:])
				if w14 {
					d00, d10 = wdec14(d00, d10)
					d01, d11 = wdec14(d01, d11)
					d00, d01 = wdec14(d00, d01)
					d10, d11 = wdec14(d10, d11)
				}
				setUint16(data[i00:], d00)
				setUint16(data[i01:], d01)
				setUint16(data[i10:], d10)
				setUint16(data[i11:], d11)
			}
		}
		m2 = m1
		m1 >>= 1
	}
}

func getUint16(bs []byte) uint16 {
	return binary.LittleEndian.Uint16(bs)
}

func setUint16(bs []byte, v uint16) {
	binary.LittleEndian.PutUint16(bs, v)
}

func wenc14(a, b uint16) (avg, dlt uint16) {
	avg = (a + b) >> 1
	dlt = a - b
	return avg, dlt
}

func wdec14(avg, dlt uint16) (a, b uint16) {
	a = avg + (dlt & 1) + (dlt >> 1)
	b = a - dlt
	return a, b
}
