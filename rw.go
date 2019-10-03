package exr

import (
	"encoding/binary"
	"fmt"
)

func newByteReader(ord binary.ByteOrder, data []byte) *byteReader {
	return &byteReader{
		ord:  ord,
		data: data,
		i:    0,
	}
}

func (w *byteReader) Uint8() uint8 {
	v := w.data[w.i]
	w.i++
	return v
}

func (w *byteReader) Uint16() uint16 {
	v := w.ord.Uint16(w.data[w.i:])
	w.i += 2
	return v
}

func (w *byteReader) Uint32() uint32 {
	v := w.ord.Uint32(w.data[w.i:])
	w.i += 4
	return v
}

func (w *byteReader) Uint64() uint64 {
	v := w.ord.Uint64(w.data[w.i:])
	w.i += 8
	return v
}

func (w *byteReader) Bytes(n int) []byte {
	v := w.data[w.i : w.i+n]
	w.i += n
	return v
}

type byteWriter struct {
	ord  binary.ByteOrder
	data []byte
	i    int
}

func newByteWriter(ord binary.ByteOrder, data []byte) *byteWriter {
	return &byteWriter{
		ord:  ord,
		data: data,
		i:    0,
	}
}

func (w *byteWriter) Uint8(v uint8) {
	w.data[w.i] = v
	w.i++
}

func (w *byteWriter) Uint16(v uint16) {
	w.ord.PutUint16(w.data[w.i:], v)
	w.i += 2
}

func (w *byteWriter) Uint32(v uint32) {
	w.ord.PutUint32(w.data[w.i:], v)
	w.i += 4
}

func (w *byteWriter) Uint64(v uint64) {
	w.ord.PutUint64(w.data[w.i:], v)
	w.i += 8
}

func (w *byteWriter) Bytes(bs []byte) {
	copy(w.data[w.i:w.i+len(bs)], bs)
	w.i += len(bs)
}

type byteReader struct {
	ord  binary.ByteOrder
	data []byte
	i    int
}

type bitReader struct {
	data []byte
	i    int // current bit index
	n    int // number of bits in data
}

func newBitReader(data []byte, n int) *bitReader {
	return &bitReader{
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
func (r *bitReader) Read(n int) []byte {
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

func (r *bitReader) Seek(to int) {
	r.i = to
}

// Remain returns number of remaining bits in the reader.
// If it reads more than it have, it will return a negative number.
func (r *bitReader) Remain() int {
	return r.n - r.i
}

// Writer is bit writer.
type bitWriter struct {
	data []byte
	i    int
	n    int
}

// NewWriter returns a new Writer.
func newBitWriter(n int) *bitWriter {
	nbyte := n / 8
	if n%8 != 0 {
		nbyte++
	}
	data := make([]byte, nbyte)
	return &bitWriter{
		data: data,
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
func (w *bitWriter) Write(n int, bs []byte) {
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
func (w *bitWriter) Data() []byte {
	return w.data
}

// Index returns current cursor index of writer.
func (w *bitWriter) Index() int {
	return w.i
}

// Remain returns number of remaining bits in the writer.
func (w *bitWriter) Remain() int {
	return w.n - w.i
}
