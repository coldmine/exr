package main

// huffman code value is stored in first 58bit,
// and it's length store in last 6bit. (total 64bit)

// huffmanCodeValue gets the code's value part.
func huffmanCodeValue(code int64) int64 {
	return code >> 6
}

// huffmanCodeLength gets the code's length part.
func huffmanCodeLength(code int64) int64 {
	return code & 0b111111
}
