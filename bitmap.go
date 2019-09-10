package exr

const (
	USHORT_RANGE = 1 << 16
	BITMAP_SIZE  = USHORT_RANGE >> 3
)

// bitmap is used instead of []bool because it is more efficient.
// Why do we use bitmap instead of direct for-loop for data lookup?
// It has a side effect let the final lut is organized as ordered.

func bitmapFromData(data []int16) ([]byte, int, int) {
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

func forwardLutFromBitmap(bitmap []byte) ([]int16, int) {
	lut := make([]int16, USHORT_RANGE)
	k := 0
	for d := range lut {
		hasD := (bitmap[d>>3] & (1 << (d & 0b111))) != 0
		if d == 0 || hasD {
			lut[d] = int16(k)
			k++
		} else {
			lut[d] = 0
		}
	}
	max := k - 1
	return lut, max
}

func reverseLutFromBitmap(bitmap []byte) ([]int16, int) {
	lut := make([]int16, USHORT_RANGE)
	k := 0
	for d := range lut {
		hasD := (bitmap[d>>3] & (1 << (d & 0b111))) != 0
		if d == 0 || hasD {
			lut[k] = int16(d)
			k++
		}
	}
	max := k - 1
	return lut, max
}

func applyLut(data []int16, lut []int16) {
	for i := range data {
		data[i] = lut[data[i]]
	}
}
