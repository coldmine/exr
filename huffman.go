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

	// caculate the data min and max before adding a pseudo symbol.
	dMin := data[0]
	dMax := data[len(data)-1]

	// add a pseudo symbol for run-length encoding.
	// TODO: what does this do?
	dMax++
	freq[dMax] = 1
	data = append(data, dMax)

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

// huffmanLengthPacksFromCodes returns packs (that only contains length part) from the codes.
func huffmanLengthPacksFromCodes(codes []int64, iMin, iMax int) []int64 {
	packs := make([]int64, 0)
	for i := iMin; i < iMax; i++ {
		l := huffmanCodeLength(codes[i])
		if l != 0 {
			packs = append(packs, l)
			continue
		}
		// zero
		n := 1
		// continuous zeros will be compressed
		// n  | length
		// ---|--------------------
		// 1  | 0
		// 2  | 59
		// 3  | 60
		// 4  | 61
		// 5  | 62
		// 6+ | 63, n-6 (2 packs)
		for i < iMax && n < (255+6) {
			if huffmanCodeLength(codes[i]) != 0 {
				break
			}
			i++
			n++
		}
		if n == 1 {
			packs = append(packs, 0)
		} else if n <= 5 {
			packs = append(packs, int64(n+57))
		} else {
			packs = append(packs, 63)
			packs = append(packs, int64(n-6))
		}
	}
	return packs
}
