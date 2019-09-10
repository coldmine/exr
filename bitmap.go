package exr

const (
	DATA_RANGE  = 1 << 16
	BITMAP_SIZE = DATA_RANGE / 8
)

// bitmap is gather information each data existing with []byte
// instead of []bool because it is more memory efficient.
//
// If it was []bool, we could do `bitmap[i] = true` to show i exist
// in the data.
// Now we should do `bitmap[i >> 3] = i & b111` instead.
//
// Why do we use bitmap instead of direct for-loop to generate lut?
// It has a effect let the lut is ordered (value increased as data increased).

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

func applyLut(data []uint16, lut []uint16) {
	for i := range data {
		data[i] = lut[data[i]]
	}
}
