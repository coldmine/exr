package exr

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
