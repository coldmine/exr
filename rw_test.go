package exr

import (
	"reflect"
	"testing"
)

func TestBitReader(t *testing.T) {
	r := newBitReader([]byte{
		0b00000000,
		0b11111111,
		0b00001111,
		0b00110011,
		0b01010101,
	}, 40)
	cases := []struct {
		nreads []int
		want   [][]byte
	}{
		{
			nreads: []int{6, 6, 6, 6, 6, 6, 4},
			want: [][]byte{
				[]byte{0b00000000},
				[]byte{0b00111100},
				[]byte{0b11110000},
				[]byte{0b00111100},
				[]byte{0b00110000},
				[]byte{0b11010100},
				[]byte{0b01010000},
			},
		},
		{
			nreads: []int{4, 8, 12, 16},
			want: [][]byte{
				[]byte{0b00000000},
				[]byte{0b00001111},
				[]byte{0b11110000, 0b11110000},
				[]byte{0b00110011, 0b01010101},
			},
		},
		{
			// read more than the reader have
			nreads: []int{50},
			want: [][]byte{
				[]byte{
					0b00000000,
					0b11111111,
					0b00001111,
					0b00110011,
					0b01010101,
					0b00000000,
					0b00000000,
				},
			},
		},
	}
	for i, c := range cases {
		r.Seek(0)
		got := make([][]byte, 0)
		for _, n := range c.nreads {
			got = append(got, r.Read(n))
		}
		n := r.Remain()
		if n != 0 {
			t.Fatalf("number of remaining bits should be 0, got %d", n)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got[%d]: %v, want: %v", i, got, c.want)
		}
	}
}

type bitSlice struct {
	n int
	b []byte
}

func TestBitWriter(t *testing.T) {
	cases := []struct {
		bits []bitSlice
		want []byte
	}{
		{
			bits: []bitSlice{
				{6, []byte{0b00000000}},
				{6, []byte{0b00111100}},
				{6, []byte{0b11110000}},
				{6, []byte{0b00111100}},
				{6, []byte{0b00110000}},
				{6, []byte{0b11010100}},
				{4, []byte{0b01010000}},
			},
			want: []byte{
				0b00000000,
				0b11111111,
				0b00001111,
				0b00110011,
				0b01010101,
			},
		},
		{
			bits: []bitSlice{
				{4, []byte{0b00000000}},
				{8, []byte{0b00001111}},
				{12, []byte{0b11110000, 0b11110000}},
				{16, []byte{0b00110011, 0b01010101}},
			},
			want: []byte{
				0b00000000,
				0b11111111,
				0b00001111,
				0b00110011,
				0b01010101,
			},
		},
		{
			bits: []bitSlice{
				{40, []byte{
					0b00000000,
					0b11111111,
					0b00001111,
					0b00110011,
					0b01010101,
				}},
			},
			want: []byte{
				0b00000000,
				0b11111111,
				0b00001111,
				0b00110011,
				0b01010101,
			},
		},
	}
	for i, c := range cases {
		w := newBitWriter(40)
		for _, b := range c.bits {
			w.Write(b.n, b.b)
		}
		n := w.Remain()
		if n != 0 {
			t.Fatalf("number of remaining bits should be 0, got %d", n)
		}
		got := w.Data()
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("got[%d] %v, want %v", i, got, c.want)
		}
	}
}
