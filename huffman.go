package exr

import "container/heap"

const (
	HUF_ENCBITS = 16
	HUF_ENCSIZE = (1 << HUF_ENCBITS) + 1
)

// A huffman code and it's length packed in 64 bits.
// So, I will call this a pack.

// huffmanCode gets the code from a pack. (first 58 bits)
func huffmanCode(pack int64) int64 {
	return pack >> 6
}

// huffmanCodeLength gets the code's length from a pack. (last 6 bits)
func huffmanCodeLength(pack int64) int64 {
	return pack & 0b111111
}

// huffmanBuildCanonicalCodes build canonical huffman codes
// from huffman code lengths.
//
// see `Code Construction` part of http://www.compressconsult.com/huffman/
//
// packs should only having the length parts (lower 6 bits) when given.
// It will assign their codes using the lengths.
func huffmanBuildCanonicalCodes(packs []int64) {
	// length of packs shold be HUF_ENCSIZE
	if len(packs) != HUF_ENCSIZE {
		panic("length of packs are not HUF_ENCSIZE")
	}
	// check how many codes are exist in each length.
	freq := make([]int64, 59)
	for i := range packs {
		l := packs[i]
		freq[l] += 1
	}
	c := int64(0)
	// calculate start code of each length.
	startCode := make([]int64, 59)
	for i := 58; i > 0; i-- {
		startCode[i] = c
		c = (c + freq[i]) >> 1
	}
	// assign codes to packs
	for i := range packs {
		l := packs[i]
		if l > 0 {
			packs[i] = l | (startCode[i] << 6)
			startCode[i]++
		}
	}
}

func huffmanCountFrequencies(data []int16) []int64 {
	freq := make([]int64, HUF_ENCSIZE)
	for _, d := range data {
		freq[d]++
	}
	return freq
}

type indexHeap struct {
	data  []int
	value func(d int) int64
}

func newIndexHeap(data []int, value func(d int) int64) indexHeap {
	return indexHeap{
		data:  data,
		value: value,
	}
}

func (h indexHeap) Len() int {
	return len(h.data)
}

func (h indexHeap) Less(i, j int) bool {
	return h.value(h.data[i]) < h.value(h.data[j])
}

func (h indexHeap) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
}

func (h indexHeap) Push(v interface{}) {
	h.data = append(h.data, v.(int))
}

func (h indexHeap) Pop() interface{} {
	n := len(h.data)
	v := h.data[n-1]
	h.data = h.data[:n-1]
	return v
}

func huffmanBuildCodes(freq []int64) ([]int64, int, int) {
	// get data those frequency is non-zero.
	data := make([]int, 0, HUF_ENCSIZE)
	for d := 0; d < HUF_ENCSIZE; d++ {
		// freq's index is data itself.
		f := freq[d]
		if f != 0 {
			data = append(data, d)
		}
	}

	// hlink creates internal nodes in a memory efficient way.
	// hlink[i] indicates the next item in hlink.
	// hlink[i] == j, hlink[j] == k ... and so on.
	// (i, j, k here doesn't mean that they are numerically continuous.)
	// if the links reached the end, say z, hlink[z] == z
	hlink := make([]int, HUF_ENCSIZE)
	for _, d := range data {
		f := freq[d]
		if f != 0 {
			hlink[d] = d
		}
	}

	// add a pseudo symbol for run-length encoding.
	// TODO: what does this do?
	symbol := data[len(data)-1] + 1
	freq[symbol] = 1
	data = append(data, symbol)

	// get min and max data before they are mixed by heap.
	dMin := data[0]
	dMax := symbol

	// create a index heap that can access to the frequency of data.
	freqHeap := newIndexHeap(data, func(d int) int64 {
		return freq[d]
	})
	heap.Init(freqHeap)

	// each pack will get the length of code for data d.
	packs := make([]int64, HUF_ENCSIZE)
	n := len(data)
	for n > 1 {
		// pop two least seen data, merge, push it back.
		n--
		a := heap.Pop(freqHeap).(int)
		b := heap.Pop(freqHeap).(int)
		fsum := freq[a] + freq[b]
		freq[a] = fsum
		freq[b] = 0
		heap.Push(freqHeap, a)

		// merge a and b's links too.
		// we need this to calculate length of codes.
		merged := false
		for d := a; ; {
			// increase length of the code
			packs[d]++

			if hlink[d] == d {
				// we will reach here twice, when a or b's links end.
				if !merged {
					// a's links end.
					// merge b, then follow b's link.
					hlink[d] = b
				} else {
					// b's links end.
					// done.
					break
				}
			}
			d = hlink[d] // follow the link
		}
	}

	// we've done calculating code length for each data.
	// assign canonical codes to the lengths.
	huffmanBuildCanonicalCodes(packs)
	return packs, dMin, dMax
}

type bitReader struct {
	data []byte
	// remain indicates how many bits are remain
	// in the first byte of data.
	remain int
}

func newBitReader(data []byte) *bitReader {
	return &bitReader{
		data:   data,
		remain: 8,
	}
}

func (r *bitReader) Read6() byte { return r.read(6) }

func (r *bitReader) Read8() byte { return r.read(8) }

func (r *bitReader) read(n int) byte {
	if r.remain == 0 {
		r.remain += 8
		r.data = r.data[1:]
	}
	// see how many bits we can read in the first byte
	nr := n
	if r.remain < n {
		nr = r.remain
	}
	unread := n - nr
	// read
	b := r.data[0] | ((1 << r.remain) - 1)
	// recalculate remaining bits in the first byte
	r.remain -= nr
	if r.remain == 0 && unread > 0 {
		r.remain += 8
		r.data = r.data[1:]
		// read unread bits
		b = (b << unread) | (r.data[0] >> (8 - unread))
	}
	return b
}

type bitWriter struct {
	data []byte
	// remain indicates how many bits are remain to write
	// in the first byte of data.
	remain int
}

func newBitWriter(data []byte) *bitWriter {
	return &bitWriter{
		data:   make([]byte, 0),
		remain: 8,
	}
}

func (w *bitWriter) Write6(b byte) { w.write(6, b) }

func (w *bitWriter) Write8(b byte) { w.write(8, b) }

func (w *bitWriter) write(n int, b byte) {
	if w.remain == 0 {
		w.remain += 8
		w.data = append(w.data, 0)
	}
	// see how many bits we can write in the first byte
	nw := n
	if w.remain < n {
		nw = w.remain
	}
	unwritten := n - nw
	// write bits
	w.data[0] |= (b >> unwritten)
	// recalcuate remaining bits in the first byte
	w.remain -= nw
	if w.remain == 0 && unwritten > 0 {
		w.remain += 8
		w.data = append(w.data, 0)
		// write unwritten bits
		w.data[0] |= (b << (8 - unwritten))
	}
}

// huffmanEncodePack encodes input packs to bits.
// Note that bits is []byte type, but grouped in 6 bits usually,
// except when containing 6+ zeros. (6 + 8 bits)
func huffmanEncodePack(data []byte, iMin, iMax int) []byte {
	w := newBitWriter(data)
	packs := make([]int64, 0)
	for i := iMin; i < iMax; i++ {
		l := huffmanCodeLength(packs[i])
		if l != 0 {
			packs = append(packs, l)
			continue
		}
		// zero
		n := 1
		// compress continuous zeros
		// n  | huffman code length
		// ---|----------------------
		// 1  | 0
		// 2  | 59
		// 3  | 60
		// 4  | 61
		// 5  | 62
		// 6+ | 63, n-6  (6 + 8 bits)
		for i < iMax && n < (255+6) {
			if huffmanCodeLength(packs[i]) != 0 {
				break
			}
			i++
			n++
		}
		if n == 1 {
			w.Write6(0)
		} else if n <= 5 {
			w.Write6(byte(int(n) + 57))
		} else {
			w.Write6(byte(63))
			w.Write8(byte(n - 6))
		}
	}
	return w.data
}

// huffmanDecodePack returns packs from the bits that contains length info.
// Note that bits is []byte type, but grouped in 6 bits usually,
// except when containing 6+ zeros. (6 + 8 bits)
func huffmanDecodePack(data []byte, dMin, dMax int) []int64 {
	r := newBitReader(data)
	packs := make([]int64, HUF_ENCSIZE)
	for d := dMin; d < dMax; d++ {
		l := int(r.Read6())
		packs[d] = int64(l)
		// decompress continuous zeros
		// n  | huffman code length
		// ---|----------------------
		// 1  | 0
		// 2  | 59
		// 3  | 60
		// 4  | 61
		// 5  | 62
		// 6+ | 63, n-6  (6 + 8 bits)
		if l >= 59 {
			var n int
			if l == 63 {
				n = int(r.Read8()) + 6
			} else {
				n = l - 59 + 2
			}
			for n != 0 {
				packs[d] = 0
				d++
			}
			d--
		}
	}
	huffmanBuildCanonicalCodes(packs)
	return packs
}
