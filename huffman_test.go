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
	})
	nReads := []int{6, 6, 6, 6, 6, 6, 4}
	want := []byte{
		0b000000,
		0b001111,
		0b111100,
		0b001111,
		0b001100,
		0b110101,
		0b0101,
	}
	got := make([]byte, len(nReads))
	for i, n := range nReads {
		g := r.Read(n)
		got[i] = g
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestBitWriter(t *testing.T) {
	w := newBitWriter()
	data := []struct {
		n int
		b byte
	}{
		{6, 0b000000},
		{6, 0b001111},
		{6, 0b111100},
		{6, 0b001111},
		{6, 0b001100},
		{6, 0b110101},
		{4, 0b0101},
	}
	for _, d := range data {
		w.Write(d.n, d.b)
	}
	got := w.Data()
	want := []byte{
		0b00000000,
		0b11111111,
		0b00001111,
		0b00110011,
		0b01010101,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
