package exr

const (
	HUF_ENCBITS = 16
	HUF_ENCSIZE = (1 << HUF_ENCBITS) + 1
)

// A huffman code and it's length packed in 64 bits.
// So, I will call this a pack.

// huffmanCode gets the code from a pack. (first 58 bits)
func huffmanCode(pack int64) int64 {
	return pack >> 6
}

// huffmanCodeLength gets the code's length from a pack. (last 6 bits)
func huffmanCodeLength(pack int64) int64 {
	return pack & 0b111111
}

// huffmanBuildCanonicalCodes build canonical huffman codes
// from huffman code lengths.
//
// see `Code Construction` part of http://www.compressconsult.com/huffman/
//
// packs should only having the length parts (lower 6 bits) when given.
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

// huffmanLengthPacksFromCodes returns packs (that only contains length part) from the codes.
func huffmanLengthPacksFromCodes(codes []int64, iMin, iMax int) []int64 {
	packs := make([]int64, 0)
	for i := iMin; i < iMax; i++ {
		l := huffmanCodeLength(codes[i])
		if l != 0 {
			packs = append(packs, l)
			continue
		}
		// zero
		n := 1
		// continuous zeros will be compressed
		// n  | length
		// ---|--------------------
		// 1  | 0
		// 2  | 59
		// 3  | 60
		// 4  | 61
		// 5  | 62
		// 6+ | 63, n-6 (2 packs)
		for i < iMax && n < (255+6) {
			if huffmanCodeLength(codes[i]) != 0 {
				break
			}
			i++
			n++
		}
		if n == 1 {
			packs = append(packs, 0)
		} else if n <= 5 {
			packs = append(packs, int64(n+57))
		} else {
			packs = append(packs, 63)
			packs = append(packs, int64(n-6))
		}
	}
	return packs
}
