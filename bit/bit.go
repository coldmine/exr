package bit

import (
	"fmt"
)

type Reader struct {
	data []byte
	i    int // current bit index
	n    int // number of bits in data
}

func NewReader(data []byte, n int) *Reader {
	return &Reader{
		data: data,
		i:    0,
		n:    n,
	}
}

// Read reads n bits from the Reader, and returns it as []byte form.
//
// If n == 0, it will return empty slice.
// If n % 8 != 0, returned slice's last byte contains left-aligned bits.
// For example, if n == 9, returned slices is [oooooooo oxxxxxxx]
// where o is meaningful, and x is meaningless bit. (all x should be 0)
// If n is larger than number of remaining bits, still it will return
// bytes for n bits with unreadable bits are filled with 0.
//
// Reader's Remain function can be used to validate returned data.
func (r *Reader) Read(n int) []byte {
	if n < 0 {
		panic(fmt.Sprintf("invalid number of bits to read: %d", n))
	}
	if n == 0 {
		return []byte{}
	}
	// nhead is number of heading bits to clip.
	nhead := r.i % 8
	// buf is the reader's data zone we are interested.
	min := r.i / 8
	r.i += n
	if r.i > r.n || r.i > len(r.data)*8 {
		r.i = r.n
	}
	max := r.i / 8
	if r.i%8 != 0 {
		max++
	}
	if max > len(r.data) {
		max = len(r.data)
	}
	buf := r.data[min:max]
	// buf is shifted (unless nhead is 0),
	// un-shift while writing it to output bytes.
	nout := n / 8
	if n%8 != 0 {
		nout++
	}
	out := make([]byte, nout)
	c := buf[0] << nhead
	for i := range out {
		b := byte(0)
		if i+1 < len(buf) {
			b = buf[i+1]
		}
		out[i] = c | b>>(8-nhead)
		c = b << nhead
	}
	// remove trailing bits in the last byte
	if n%8 != 0 {
		ntrail := 8 - (n % 8)
		c = out[len(out)-1]
		out[len(out)-1] = (c >> ntrail) << ntrail
	}
	return out
}

func (r *Reader) Seek(to int) {
	r.i = to
}

// Remain returns number of remaining bits in the reader.
// If it reads more than it have, it will return a negative number.
func (r *Reader) Remain() int {
	return r.n - r.i
}

// Writer is bit writer.
type Writer struct {
	data []byte
	i    int
	n    int
}

// NewWriter returns a new Writer.
func NewWriter(n int, data []byte) *Writer {
	nbyte := n / 8
	if n%8 != 0 {
		nbyte++
	}
	if nbyte > len(data) {
		panic("data buffer is smaller than n needs")
	}
	return &Writer{
		data: data[:nbyte],
		i:    0,
		n:    n,
	}
}

// Write writes n bits to Writer.
//
// If n == 0, it does nothing.
// If n < 0, it panics.
// If n is greater than the writer's remaining buffer, it will panic.
// If n is greater than size of input bytes, it will panic.
// If n % 8 != 0, trailing bytes and bits are discarded.
//
// Writer's Remain function can be used to check number of bits are remaining
// in the writer's data buffer.
func (w *Writer) Write(n int, bs []byte) {
	if n == 0 {
		return
	}
	if n < 0 {
		panic(fmt.Sprintf("invalid number of bits to write: %d", n))
	}
	if n > len(bs)*8 {
		panic(fmt.Sprintf("tried to write more bits than the input bytes offer"))
	}
	if n > w.n-w.i {
		panic(fmt.Sprintf("tried to write more bits than the writer can have"))
	}
	// discard trailing bytes and bits which is not for this writing
	nbyte := n / 8
	nend := n % 8
	if nend != 0 {
		nbyte++
	}
	bs = bs[:nbyte]
	if nend != 0 {
		ntrail := 8 - nend
		bs[len(bs)-1] = (bs[len(bs)-1] >> ntrail) << ntrail
	}
	// write input bytes
	i := w.i
	for _, b := range bs {
		nhead := i % 8
		w.data[i/8] |= b >> nhead
		if (i/8)+1 < len(w.data) {
			w.data[(i/8)+1] = b << (8 - nhead)
		}
		i += 8
	}
	w.i = w.i + n
}

// Data returns the writer's data written so far.
func (w *Writer) Data() []byte {
	return w.data
}

// Remain returns number of remaining bits in the writer.
func (w *Writer) Remain() int {
	return w.n - w.i
}
