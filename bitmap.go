package exr

const (
	DATA_RANGE = 1 << 16
)

// bitmap shows whether each number in 0 - DATA_RANGE is exist,
// in a memory efficient way.
type bitmap []byte

func newBitmap() bitmap {
	b := make([]byte, DATA_RANGE/8)
	return b
}

func (b bitmap) Set(i uint16) {
	b[i>>3] |= 1 << (i & 0b111)
}

func (b bitmap) Unset(i uint16) {
	if !b.Has(i) {
		return
	}
	b[i>>3] -= 1 << (i & 0b111)
}

func (b bitmap) Has(i uint16) bool {
	return (b[i>>3] & (1 << (i & 0b111))) != 0
}

func (b bitmap) MinByteIndex(i uint16) int {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return i
		}
	}
	return len(b) - 1
}

func (b bitmap) MaxByteIndex(i uint16) int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return i
		}
	}
	return 0
}

// bitmapFromData creates a new bitmap from data.
func bitmapFromData(data []uint16) bitmap {
	b := newBitmap()
	for _, d := range data {
		b.Set(d)
	}
	// zero is explicitly not stored into this bitmap
	b.Unset(0)
	return b
}

// forwardLutFromBitmap returns a lut and it's max value.
// The lut maps a data number to a incremental number.
func forwardLutFromBitmap(b bitmap) ([]uint16, int) {
	lut := make([]uint16, DATA_RANGE)
	k := 0
	for d := range lut {
		if d == 0 || b.Has(uint16(d)) {
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
func reverseLutFromBitmap(b bitmap) ([]uint16, int) {
	lut := make([]uint16, DATA_RANGE)
	k := 0
	for d := range lut {
		if d == 0 || b.Has(uint16(d)) {
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
