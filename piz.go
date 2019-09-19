package exr

// compressed data struct for piz
//
// [
// 	minimum non-zero bite index of bitmap
// 	maximum non-zero byte index of bitmap
// 	bitmap
// 	length of compressed data
// 	compressed data
// ]

func pizCompress(raw []uint16, block blockInfo) []byte {
	compressed := make([]byte, 0, len(raw))
	// build bitmap
	bitm := newBitmap(1 << 16)
	for _, d := range raw {
		bitm.Set(d)
	}
	bitm.Unset(0) // don't include zero in bitmap
	minNonZero := bitm.MinByteIndex()
	maxNonZero := bitm.MaxByteIndex()
	parse.PutUint32(compressed[0:], uint32(minNonZero))
	parse.PutUint32(compressed[4:], uint32(maxNonZero))
	i := 8
	for _, d := range bitm[minNonZero : maxNonZero+1] {
		compressed[i] = d
		i++
	}
	// wavlet encoding
	chData := make(map[string][]uint16)
	var n, m int
	for _, ch := range block.channels {
		m += block.width * block.height * pixelSize(ch.pixelType)
		chData[ch.name] = raw[n:m]
		n = m
		// TODO: ySampling
	}
	// apply forward lut
	lut := forwardLutFromBitmap(bitm)
	applyLut(raw, lut)
	// wavelet encoding
	for range block.channels {
		// TODO: applyWaveletEncode(chData[ch.name], maxValue)
	}
	// compress
	cdata := huffmanCompress(raw, block)
	parse.PutUint32(compressed[i:], uint32(len(compressed)))
	i += 4
	for _, d := range cdata {
		compressed[i] = d
		i++
	}
	return compressed
}

func pizDecompress(compressed []byte, block blockInfo) []uint16 {
	// get bitmap info
	minNonZero := parse.Uint32(compressed[0:])
	maxNonZero := parse.Uint32(compressed[4:])
	bitm := newBitmap(1 << 16)
	copy(bitm[minNonZero:maxNonZero+1], compressed[8:])
	i := 8 + (maxNonZero - minNonZero + 1)
	// decompress
	lc := parse.Uint32(compressed[i:])
	i += 4
	cdata := compressed[i : i+lc]
	raw := huffmanDecompress(cdata, block)
	// wavlet decode each channel
	chData := make(map[string][]uint16)
	var n, m int
	for _, ch := range block.channels {
		m += block.width * block.height * pixelSize(ch.pixelType)
		chData[ch.name] = raw[n:m]
		n = m
		// TODO: ySampling
	}
	for range block.channels {
		// TODO: applyWaveletDecode(chData[ch.name], maxValue)
	}
	// apply reverse lut
	lut := reverseLutFromBitmap(bitm)
	applyLut(raw, lut)
	return raw
}
