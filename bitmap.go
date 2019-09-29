package exr

import "fmt"

const (
	DATA_RANGE = 1 << 16
)

// bitmap shows whether each number in 0 - DATA_RANGE is exist,
// in a memory efficient way.
// Because bitmap in exr is for dealing []uint16 data,
// it's length is DATA_RANGE/8 at maximum.
type bitmap []byte

// newBitmap returns a new bitmap that can hold up to n - 1.
// If n % 8 != 0, it will round up to nearest multiple of 8.
// When n is greater than DATA_RANGE, it will panic.
func newBitmap(n int) bitmap {
	if n > DATA_RANGE {
		panic(fmt.Sprintf("bitmap could not hold more than %d\n", DATA_RANGE))
	}
	nbyte := n / 8
	if n%8 != 0 {
		nbyte++
	}
	b := make([]byte, nbyte)
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

func (b bitmap) MinByteIndex() int {
	for i := 0; i < len(b); i++ {
		if b[i] != 0 {
			return i
		}
	}
	return len(b) - 1
}

func (b bitmap) MaxByteIndex() int {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return i
		}
	}
	return 0
}

// forwardLutFromBitmap returns a lut and it's max value.
// The lut maps a data number to a incremental number.
func forwardLutFromBitmap(b bitmap) []uint16 {
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
	return lut
}

// reverseLutFromBitmap returns a reverse lut and it's max index.
// The lut restores a data number from a incremental number.
func reverseLutFromBitmap(b bitmap) []uint16 {
	lut := make([]uint16, DATA_RANGE)
	k := 0
	for d := range lut {
		if d == 0 || b.Has(uint16(d)) {
			lut[k] = uint16(d)
			k++
		}
	}
	return lut
}
