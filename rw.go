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

func (r *byteReader) Uint8() uint8 {
	v := r.data[r.i]
	r.i++
	return v
}

func (r *byteReader) Uint16() uint16 {
	v := r.ord.Uint16(r.data[r.i:])
	r.i += 2
	return v
}

func (r *byteReader) Uint32() uint32 {
	v := r.ord.Uint32(r.data[r.i:])
	r.i += 4
	return v
}

func (r *byteReader) Uint64() uint64 {
	v := r.ord.Uint64(r.data[r.i:])
	r.i += 8
	return v
}

func (r *byteReader) Bytes(n int) []byte {
	v := r.data[r.i : r.i+n]
	r.i += n
	return v
}

func (r *byteReader) Remain() int {
	return len(r.data) - r.i
}

// ToBitReader returns a bitReader from byteReader r.
func (r *byteReader) ToBitReader() *bitReader {
	return &bitReader{
		data: r.data,
		i:    r.i * 8,
		n:    len(r.data) * 8,
	}
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

// Remain returns number of remaining bits in the writer.
func (w *byteWriter) Remain() int {
	return len(w.data) - w.i
}

// ToBitWriter returns a bitWriter from byteWriter r.
func (w *byteWriter) ToBitWriter() *bitWriter {
	return &bitWriter{
		data: w.data,
		i:    w.i * 8,
		n:    len(w.data) * 8,
	}
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

// ToByteReader returns a byteReader from bitReader r.
// Note: cursor index that isn't multiple of 8 will be converted
// to it's nearest upper byte index.
func (r *bitReader) ToByteReader(ord binary.ByteOrder) *byteReader {
	return &byteReader{
		ord:  ord,
		data: r.data,
		i:    (r.i + 7) / 8,
	}
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

// ToByteWriter returns a byteWriter from bitWriter r.
// Note: cursor index that isn't multiple of 8 will be converted
// to it's nearest upper byte index.
func (w *bitWriter) ToByteWriter(ord binary.ByteOrder) *byteWriter {
	return &byteWriter{
		ord:  ord,
		data: w.data,
		i:    (w.i + 7) / 8, // skip unread bits in current byte
	}
}
