package exr

import (
	"container/heap"
	"fmt"

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
			packs[i] = l | (startCode[i] << 6)
			startCode[i]++
		}
	}
}

func huffmanCountFrequencies(data []uint16) []uint64 {
	freq := make([]uint64, HUF_ENCSIZE)
	for _, d := range data {
		freq[d]++
	}
	return freq
}

type indexHeap struct {
	data  []int
	value func(d int) uint64
}

func newIndexHeap(data []int, value func(d int) uint64) indexHeap {
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

func huffmanBuildEncodingTable(freq []uint64) ([]uint64, int, int) {
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
	freqHeap := newIndexHeap(data, func(d int) uint64 {
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
// when it's enough, the code is left aligned and
// trailing bits are not that important.
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
	lit uint64

	// lits are data for long codes. (len > HUF_DECBITS)
	// otherwise it's nil.
	lits []uint64
}

// huffmanBuildDecodingTable returns a decoding table to decode huffman codes.
func huffmanBuildDecodingTable(packs []uint64, dMin, dMax uint64) hdec {
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
			ci := c << (HUF_DECBITS - l)
			pl := dec[ci]
			// fill all indice that are having the same heading bits.
			i := uint64(1) << (HUF_DECBITS - l)
			for i > 0 {
				if pl.len != 0 {
					panic("already been stored")
				}
				if len(pl.lits) != 0 {
					panic("already occupied by long code")
				}
				pl.len = l
				pl.lit = d
				ci++
				pl = dec[ci]
				i--
			}
		} else {
			// long code
			ci := c >> (l - HUF_DECBITS)
			pl := dec[ci]
			if pl.len != 0 {
				panic("already occupied by short code")
			}
			pl.lits = append(pl.lits, d)
		}
	}
	return nil
}

// huffmanPackEncodingTable encodes input packs to bits.
// Note that bits is []byte type, but grouped in 6 bits usually,
// except when containing 6+ zeros. (6 + 8 bits)
func huffmanPackEncodingTable(packs []uint64, iMin, iMax int) ([]byte, int) {
	w := bit.NewWriter(len(packs) * 64)
	for i := iMin; i < iMax; i++ {
		l := huffmanCodeLength(packs[i])
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
			if huffmanCodeLength(packs[i]) != 0 {
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
func huffmanUnpackEncodingTable(data []byte, nBits int, dMin, dMax uint64) []uint64 {
	r := bit.NewReader(data, len(data)*8)
	packs := make([]uint64, HUF_ENCSIZE)
	for d := dMin; d < dMax; d++ {
		l := int(r.Read(6)[0])
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

// readCode read codes from bit.Reader r.
func readCode(r *bit.Writer, p, runp uint64, run uint8) {
}

// huffmanEncode encodes packs to output bytes.
func huffmanEncode(raw []uint16, packs []uint64, runCode int) ([]byte, int) {
	w := bit.NewWriter(len(packs) * 8)
	// run length encoding
	var run uint8
	prev := raw[0]
	for _, c := range raw[1:] {
		if c == prev && run < 255 {
			run++
		} else {
			writeCode(w, packs[prev], packs[runCode], run)
			run = 0
		}
		prev = c
	}
	writeCode(w, packs[prev], packs[runCode], run)
	return w.Data(), w.Index()
}

// huffmanDecode decodes packs to output bytes.
func huffmanDecode(compressed []byte, nBits int, dec hdec) ([]uint16, int) {
	return nil, 0
}

// huffmanCompress compress raw channel data.
func huffmanCompress(raw []uint16, block blockInfo) []byte {
	compressed := make([]byte, len(raw)*2)
	if len(raw) == 0 {
		return compressed
	}
	freqs := huffmanCountFrequencies(raw)
	packs, dMin, dMax := huffmanBuildEncodingTable(freqs)
	packBytes, nBitsPack := huffmanPackEncodingTable(packs, dMin, dMax)
	runCode := dMax
	dataBytes, nBitsData := huffmanEncode(raw, packs, runCode)
	parse.PutUint32(compressed[0:], uint32(dMin))       // [0:4]
	parse.PutUint32(compressed[4:], uint32(dMax))       // [4:8]
	parse.PutUint32(compressed[8:], uint32(nBitsPack))  // [8:12]
	parse.PutUint32(compressed[12:], uint32(nBitsData)) // [12:16]
	// compressed[16:20] is room for future extensions
	i := 20
	for _, b := range packBytes {
		compressed[i] = b
		i++
	}
	for _, b := range dataBytes {
		compressed[i] = b
		i++
	}
	return compressed
}

func huffmanDecompress(compressed []byte, block blockInfo) []uint16 {
	dMin := uint64(parse.Uint32(compressed))
	dMax := uint64(parse.Uint32(compressed[4:]))
	nBitsPack := int(parse.Uint32(compressed[8:]))
	nBitsData := int(parse.Uint32(compressed[12:]))
	// compressed[16:20] is room for future extensions
	i := uint64(20)
	packs := huffmanUnpackEncodingTable(compressed[i:], nBitsPack, dMin, dMax)
	i += dMax - dMin + 1
	dec := huffmanBuildDecodingTable(packs, dMin, dMax)
	raw, nBits := huffmanDecode(compressed[i:], nBitsData, dec)
	pixsize := 0
	for _, ch := range block.channels {
		pixsize += pixelSize(ch.pixelType)
	}
	bufSize := block.width * block.height * pixsize
	runCode := dMax
	fmt.Println(nBits, bufSize, runCode)
	return raw
}
