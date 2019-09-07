package bit

import "fmt"

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

type Writer struct {
	data []byte
	// remain indicates how many bits are remain to write
	// in the data[i].
	remain int
}

func NewWriter() *Writer {
	return &Writer{
		data:   make([]byte, 0),
		remain: 0,
	}
}

// Write writes n (0 - 8) bits from Writer.
func (w *Writer) Write(n int, b byte) {
	if n > 8 || n < 0 {
		panic(fmt.Sprintf("invalid number of bits to write: %d", n))
	}
	if n == 0 {
		return
	}
	if w.remain <= 0 {
		w.remain += 8
		w.data = append(w.data, 0)
	}
	c := uint16(b)
	c = c << (16 - n)       // left align
	c = c >> (8 - w.remain) // shift to remaining
	w.data[len(w.data)-1] |= byte(c >> 8)
	if n > w.remain {
		w.remain += 8
		w.data = append(w.data, byte(c))
	}
	w.remain -= n
}

func (w *Writer) Data() []byte {
	return w.data
}
