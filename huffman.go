package exr

import (
	"container/heap"
	"encoding/binary"

	"github.com/coldmine/exr/bit"
)

const (
	HUF_ENCBITS = 16
	HUF_DECBITS = 14
	HUF_ENCSIZE = (1 << HUF_ENCBITS) + 1
	HUF_DECSIZE = (1 << HUF_DECBITS)
	HUF_DECMASK = (1 << HUF_DECBITS) - 1
)

// A huffman code and it's length packed in 64 bits.
// So, I will call this a pack.

// huffmanCode gets the code from a pack. (first 58 bits)
func huffmanCode(pack uint64) uint64 {
	return pack >> 6
}

// huffmanCodeLength gets the code's length from a pack. (last 6 bits)
func huffmanCodeLength(pack uint64) int {
	return int(pack & 0b111111)
}

// huffmanBuildCanonicalCodes build canonical huffman codes
// from huffman code lengths.
//
// see `Code Construction` part of http://www.compressconsult.com/huffman/
//
// packs should only having the length parts (lower 6 bits) when given.
// It will assign their codes using the lengths.
func huffmanBuildCanonicalCodes(packs []uint64) {
	// length of packs shold be HUF_ENCSIZE
	if len(packs) != HUF_ENCSIZE {
		panic("length of packs are not HUF_ENCSIZE")
	}
	// check how many codes are exist in each length.
	freq := make([]uint64, 59)
	for i := range packs {
		l := packs[i]
		freq[l] += 1
	}
	c := uint64(0)
	// calculate start code of each length.
	startCode := make([]uint64, 59)
	for i := 58; i > 0; i-- {
		startCode[i] = c
		c = (c + freq[i]) >> 1
	}
	// assign codes to packs
	for i := range packs {
		l := packs[i]
		if l > 0 {
			packs[i] = (startCode[l] << 6) | l
			startCode[l]++
		}
	}
}

func huffmanCountFrequencies(raw []byte) []int {
	freq := make([]int, HUF_ENCSIZE)
	r := newByteReader(binary.LittleEndian, raw)
	for i := 0; i < len(raw); i += 2 {
		d := r.Uint16()
		freq[d]++
	}
	return freq
}

type indexHeap struct {
	idx   []int
	value func(d int) int
}

func newIndexHeap(idx []int, value func(d int) int) indexHeap {
	return indexHeap{
		idx:   idx,
		value: value,
	}
}

func (h indexHeap) Len() int {
	return len(h.idx)
}

func (h indexHeap) Less(i, j int) bool {
	return h.value(h.idx[i]) < h.value(h.idx[j])
}

func (h indexHeap) Swap(i, j int) {
	h.idx[i], h.idx[j] = h.idx[j], h.idx[i]
}

func (h indexHeap) Push(v interface{}) {
	h.idx = append(h.idx, v.(int))
}

func (h indexHeap) Pop() interface{} {
	n := len(h.idx)
	v := h.idx[n-1]
	h.idx = h.idx[:n-1]
	return v
}

func huffmanBuildEncodingTable(freq []int) ([]uint64, int, int) {
	// get data those frequency is non-zero.
	data := make([]int, 0, HUF_ENCSIZE)
	for d := 0; d < HUF_ENCSIZE; d++ {
		if freq[d] != 0 {
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
	freqHeap := newIndexHeap(data, func(d int) int {
		return freq[d]
	})
	heap.Init(freqHeap)

	// each pack will get the length of code for data d.
	packs := make([]uint64, HUF_ENCSIZE)
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
					merged = true
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

// hdec is a decoding table for efficient huffman decoding.
//
// index of hdec indicates the heading bits of a code.
// these heading bits may or may not enough to store the code.
// when it's enough, only n-bits on the left are meaningful.
// (n is length of the code)
// when it's not enough, all data from codes having
// the heading bits are stored to hdec[d].lits
//
// see http://www.compressconsult.com/huffman/#decoding
type hdec []struct {
	// len specifies length of a code.
	// when it is a short code, it is length of the code.
	// otherwise it's 0.
	len int

	// lit is data for a short code. (len <= HUF_DECBITS)
	// otherwise it's 0.
	lit int

	// lits are data for long codes. (len > HUF_DECBITS)
	// otherwise it's nil.
	lits []int
}

// huffmanBuildDecodingTable returns a decoding table to decode huffman codes.
func huffmanBuildDecodingTable(packs []uint64, dMin, dMax int) hdec {
	dec := make(hdec, HUF_DECSIZE)
	for d := dMin; d <= dMax; d++ {
		c := huffmanCode(packs[d])
		l := huffmanCodeLength(packs[d])
		if c>>l != 0 {
			panic("code didn't match to it's length")
		}
		if l == 0 {
			continue
		} else if l <= HUF_DECBITS {
			// short code
			i := c << (HUF_DECBITS - l)
			// fill all indice that are having the same heading bits.
			n := uint64(1) << (HUF_DECBITS - l)
			for n > 0 {
				if dec[i].len != 0 {
					panic("already been stored")
				}
				if len(dec[i].lits) != 0 {
					panic("already occupied by long code")
				}
				dec[i].len = l
				dec[i].lit = d
				i++
				n--
			}
		} else {
			// long code
			i := c >> (l - HUF_DECBITS)
			if dec[i].len != 0 {
				panic("already occupied by short code")
			}
			dec[i].lits = append(dec[i].lits, d)
		}
	}
	return dec
}

// huffmanPackEncodingTable encodes input packs to bits.
// Note that bits is []byte type, but grouped in 6 bits usually,
// except when containing 6+ zeros. (6 + 8 bits)
func huffmanPackEncodingTable(packs []uint64, iMin, iMax int) ([]byte, int) {
	w := bit.NewWriter(len(packs) * 64)
	for i := iMin; i < iMax; i++ {
		l := huffmanCodeLength(packs[uint64(i)])
		if l != 0 {
			packs = append(packs, uint64(l))
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
			if huffmanCodeLength(packs[uint64(i)]) != 0 {
				break
			}
			i++
			n++
		}
		if n == 1 {
			w.Write(6, []byte{0})
		} else if n <= 5 {
			w.Write(6, []byte{byte(n + 57)})
		} else {
			w.Write(6, []byte{63})
			w.Write(8, []byte{byte(n - 6)})
		}
	}
	return w.Data(), w.Index()
}

// huffmanUnpackEncodingTable returns packs from the bits that contains length info.
// Note that bits is []byte type, but grouped in 6 bits usually,
// except when containing 6+ zeros. (6 + 8 bits)
func huffmanUnpackEncodingTable(bs []byte, dMin, dMax int) []uint64 {
	r := bit.NewReader(bs, len(bs)*8)
	packs := make([]uint64, HUF_ENCSIZE)
	for d := dMin; d <= dMax; d++ {
		l := int(r.Read(6)[0] >> 2)
		packs[d] = uint64(l)
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
				n = int(r.Read(8)[0]) + 6
			} else {
				n = l - 59 + 2
			}
			for n != 0 {
				packs[d] = 0
				d++
				n--
			}
			d--
		}
	}
	huffmanBuildCanonicalCodes(packs)
	return packs
}

func uint64ToBytes(n uint64) []byte {
	bs := make([]byte, 8)
	parse.PutUint64(bs, n)
	return bs
}

// writeCode writes codes to bit.Writer w.
// It uses run length encoding when they are shorter than normal encoding.
func writeCode(w *bit.Writer, p, runp uint64, run uint8) {
	n := huffmanCodeLength(p) * int(run)
	nrun := huffmanCodeLength(p) + huffmanCodeLength(runp) + 8
	if nrun < n {
		w.Write(huffmanCodeLength(p), uint64ToBytes(huffmanCode(p)))
		w.Write(huffmanCodeLength(runp), uint64ToBytes(huffmanCode(runp)))
		w.Write(8, []byte{run})
	} else {
		for i := 0; i < int(run); i++ {
			w.Write(huffmanCodeLength(p), uint64ToBytes(huffmanCode(p)))
		}
	}
}

// huffmanEncode encodes packs to output bytes.
func huffmanEncode(raw []byte, packs []uint64, runCode int) ([]byte, int) {
	r := newByteReader(binary.LittleEndian, raw)
	w := bit.NewWriter(len(packs) * 8)
	// run length encoding
	var run uint8
	prev := r.Uint16()
	for i := 2; i < len(raw); i += 2 {
		c := r.Uint16()
		if c == prev && run < 255 {
			run++
		} else {
			writeCode(w, packs[prev], packs[uint64(runCode)], run)
			run = 0
		}
		prev = c
	}
	writeCode(w, packs[prev], packs[uint64(runCode)], run)
	return w.Data(), w.Index()
}

// huffmanDecode decodes packs to output bytes.
func huffmanDecode(block blockInfo, data []byte, nBits int, dec hdec, packs []uint64, runCode int) []byte {
	raw := make([]byte, block.pixsize)
	w := newByteWriter(binary.LittleEndian, raw)
	r := bit.NewReader(data, nBits)
	c := uint64(0)
	lc := 0
READ:
	for {
		// read until c is full or reader is run out of bits
		nr := r.Remain()
		if nr == 0 {
			break
		}
		if nr > (64 - lc) {
			nr = 64 - lc
		}
		nhead := nr % 8
		if nhead != 0 {
			nr -= nhead
			c = (c << nhead) | uint64(r.Read(nhead)[0]>>(8-nhead))
			lc += nhead
		}
		bs := r.Read(nr)
		for _, b := range bs {
			c = (c << 8) | uint64(b)
			lc += 8
		}
		// process
		for lc >= HUF_DECBITS {
			pl := dec[(c>>(lc-HUF_DECBITS))&HUF_DECMASK]
			l := 0
			if pl.len != 0 {
				// short code
				w.Uint16(uint16(packs[pl.lit]))
			} else {
				// long code
				found := false
				for _, d := range pl.lits {
					l = huffmanCodeLength(packs[d])
					if l > 64 {
						panic("code length should not bigger than 64")
					}
					if l > lc {
						continue READ
					}
					code := (c >> (lc - l)) & ((1 << l) - 1)
					if huffmanCode(packs[d]) == code {
						found = true
						w.Uint16(uint16(d))
						break
					}
				}
				if !found {
					panic("long code not found")
				}
			}
			c = (c << (64 - l)) >> (64 - l)
			lc -= l
		}
	}
	if lc != 0 {
		// TODO: handle it?
	}
	return raw
}

// huffmanCompress compress raw channel data.
func huffmanCompress(block blockInfo, raw []byte) []byte {
	compressed := make([]byte, len(raw))
	w := newByteWriter(binary.LittleEndian, compressed)
	freqs := huffmanCountFrequencies(raw)
	packs, dMin, dMax := huffmanBuildEncodingTable(freqs)
	packBytes, nBitsPack := huffmanPackEncodingTable(packs, dMin, dMax)
	runCode := dMax
	rawBytes, nBitsData := huffmanEncode(raw, packs, runCode)
	w.Uint32(uint32(dMin))
	w.Uint32(uint32(dMax))
	w.Uint32(uint32(nBitsPack))
	w.Uint32(uint32(nBitsData))
	w.Uint32(0) // compressed[16:20] is room for future extensions
	for _, b := range packBytes {
		w.Uint8(b)
	}
	for _, b := range rawBytes {
		w.Uint8(b)
	}
	return compressed
}

func huffmanDecompress(block blockInfo, compressed []byte) []byte {
	r := newByteReader(binary.LittleEndian, compressed)
	dMin := int(r.Uint32())
	dMax := int(r.Uint32())
	_ = int(r.Uint32()) // tableLength
	nBitsData := int(r.Uint32())
	_ = r.Uint32() // compressed[16:20] is room for future extensions

	packs := huffmanUnpackEncodingTable(compressed[20:], dMin, dMax)
	// TODO: r must shifted as much huffmanUnpackEncodingTable reads
	dec := huffmanBuildDecodingTable(packs, dMin, dMax)

	data := r.Bytes((nBitsData + 7) / 8)
	runCode := dMax
	return huffmanDecode(block, data, nBitsData, dec, packs, runCode)
}
