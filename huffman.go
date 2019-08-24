package exr

const (
	HUF_ENCBITS = 16
	HUF_ENCSIZE = (1 << HUF_ENCBITS) + 1
)

// A huffman code and it's length packed in 64 bit.
// So, I will call this a pack.

// huffmanCode gets the code from a pack. (first 58 bit)
func huffmanCode(pack int64) int64 {
	return pack >> 6
}

// huffmanCodeLength gets the code's length from a pack. (last 6 bit)
func huffmanCodeLength(pack int64) int64 {
	return pack & 0b111111
}

// huffmanBuildCanonicalCodes build canonical huffman codes
// from huffman code lengths.
//
// see `Code Construction` part of http://www.compressconsult.com/huffman/
//
// packs should only having the length parts (lower 6 bit) when given.
// It will assign their codes using the lengths.
func huffmanBuildCanonicalCodes(packs []int64) {
	// length of packs shold be HUF_ENCSIZE
	if len(packs) != HUF_ENCSIZE {
		panic("length of packs are not HUF_ENCSIZE")
	}
	// check how many codes are exist in each length.
	freq := make([]int64, 59)
	for i := range packs {
		l := packs[i]
		freq[l] += 1
	}
	c := int64(0)
	// calculate start code of each length.
	startCode := make([]int64, 59)
	for i := 58; i > 0; i-- {
		startCode[i] = c
		c = (c + freq[i]) >> 1
	}
	// assign codes to packs
	for i := range packs {
		l := packs[i]
		if l > 0 {
			packs[i] = l | (startCode[i] << 6)
			startCode[i]++
		}
	}
}
