package bit

import "fmt"

type Reader struct {
	data []byte
	// i is the current byte index
	i int
	// remain indicates how many bits are remain
	// in the data[i]
	remain int
}

func NewReader(data []byte) *Reader {
	return &Reader{
		data:   data,
		i:      0,
		remain: 8,
	}
}

// Read reads n (0 - 8) bits from the Reader.
// If there is no more data to read, it will return 0.
func (r *Reader) Read(n int) byte {
	if n > 8 || n < 0 {
		panic(fmt.Sprintf("invalid number of bits to read: %d", n))
	}
	if n == 0 {
		return 0
	}
	c := uint16(r.data[r.i]) << 8
	if r.i+1 < len(r.data) {
		c |= uint16(r.data[r.i+1])
	}
	c = c << (8 - r.remain) // clear unused heading bits.
	c = c >> (16 - n)       // clear unused tailing bits.
	r.remain -= n
	if r.remain <= 0 {
		r.remain += 8
		r.i += 1
	}
	return byte(c)
}

// Done checks whether the Reader has remaining data to read.
func (r *Reader) Done() bool {
	return r.i >= len(r.data)
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
