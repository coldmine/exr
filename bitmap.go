package exr

const (
	USHORT_RANGE = 1 << 16
	BITMAP_SIZE  = USHORT_RANGE >> 3
)

// bitmap is used instead of []bool because it is more efficient.
// Why do we use bitmap instead of direct for-loop for data lookup?
// It has a side effect let the final lut is organized as ordered.

func bitmapFromData(data []int16) (bitmap []byte, min, max int) {
	bitmap := make([]byte, BITMAP_SIZE)
	for _, d := range data {
		bitmap[d>>3] |= 1 << (d & 0b111)
	}
	// zero is not explictly stored to bitmap
	if bitmap[0] & 1 {
		bitmap[0]--
	}
	min := BITMAP_SIZE - 1
	max := 0
	for d := range bitmap {
		if bitmap[d] != 0 {
			if bitmap[d] < min {
				min = v
			}
			if bitmap[d] > max {
				max = v
			}
		}
	}
	return bitmap, min, max
}

func forwardLutFromBitmap(bitmap []byte) (lut []int16, max int16) {
	lut := make([]int16, USHORT_SIZE)
	k := 0
	for d := range lut {
		if d == 0 || bitmap[d>>3]&(1<<(d&0b111)) {
			lut[d] = k
			k++
		} else {
			lut[d] = 0
		}
	}
	max := k - 1
	return lut, max
}

func reverseLutFromBitmap(bitmap []byte) (lut []int16, max int16) {
	lut := make([]int16, USHORT_SIZE)
	k := 0
	for d := range lut {
		if d == 0 || bitmap[d>>3]&(1<<(d&0b111)) {
			lut[k] = d
			k++
		}
	}
	max := k - 1
	return lut, max
}

func applyLut(data []int16, lut []int16) []int16 {
	for i := range data {
		data[i] = lut[data[i]]
	}
	return data
}
