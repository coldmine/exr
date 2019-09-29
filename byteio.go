package exr

import "encoding/binary"

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
