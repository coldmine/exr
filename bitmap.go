package exr

const (
	DATA_RANGE  = 1 << 16
	BITMAP_SIZE = DATA_RANGE / 8
)

// bitmap shows whether each number in 0 - DATA_RANGE is exist on input data.
//
// Usually we could do this check with [DATA_RANGE]bool,
// but [BITMAP_SIZE]byte is more memory efficient.
//
// If it was [DATA_RANGE]bool, we could simply do `bitmap[i] = true`
// to show i exist in the data.
// Now we should do `bitmap[i >> 3] = 1 << (i & b111)` instead.

// bitmapFromData checks existence of numbers, and put them into a bitmap.
// It also returns min, max number in the bitmap.
func bitmapFromData(data []uint16) ([]byte, int, int) {
	bitmap := make([]byte, BITMAP_SIZE)
	for _, d := range data {
		bitmap[d>>3] |= 1 << (d & 0b111)
	}
	// zero is not stored to bitmap
	if (bitmap[0] & 1) != 0 {
		bitmap[0]--
	}
	min := BITMAP_SIZE - 1
	max := 0
	for i, v := range bitmap {
		if v != 0 {
			if i < min {
				min = i
			}
			if i > max {
				max = i
			}
		}
	}
	return bitmap, min, max
}

// forwardLutFromBitmap returns a lut and it's max value.
// The lut maps a data number to a incremental number.
func forwardLutFromBitmap(bitmap []byte) ([]uint16, int) {
	lut := make([]uint16, DATA_RANGE)
	k := 0
	for d := range lut {
		hasD := (bitmap[d>>3] & (1 << (d & 0b111))) != 0
		if d == 0 || hasD {
			lut[d] = uint16(k)
			k++
		} else {
			lut[d] = 0
		}
	}
	max := k - 1
	return lut, max
}

// reverseLutFromBitmap returns a reverse lut and it's max index.
// The lut restores a data number from a incremental number.
func reverseLutFromBitmap(bitmap []byte) ([]uint16, int) {
	lut := make([]uint16, DATA_RANGE)
	k := 0
	for d := range lut {
		hasD := (bitmap[d>>3] & (1 << (d & 0b111))) != 0
		if d == 0 || hasD {
			lut[k] = uint16(d)
			k++
		}
	}
	max := k - 1
	return lut, max
}

// applyLut applies lut transform to data.
// It will change the data in place.
func applyLut(data []uint16, lut []uint16) {
	for i := range data {
		data[i] = lut[data[i]]
	}
}
