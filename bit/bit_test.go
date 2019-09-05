package bit

import (
	"reflect"
	"testing"
)

func TestReader(t *testing.T) {
	r := NewReader([]byte{
		0b00000000,
		0b11111111,
		0b00001111,
		0b00110011,
		0b01010101,
	}, 40)
	nReads := []int{6, 6, 6, 6, 6, 6, 4}
	want := [][]byte{
		[]byte{0b00000000},
		[]byte{0b00111100},
		[]byte{0b11110000},
		[]byte{0b00111100},
		[]byte{0b00110000},
		[]byte{0b11010100},
		[]byte{0b01010000},
	}
	got := make([][]byte, 0)
	for _, n := range nReads {
		got = append(got, r.Read(n))
	}
	n := r.Remain()
	if n != 0 {
		t.Fatalf("number of remaining bits should be 0, got %d", n)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWriter(t *testing.T) {
	w := NewWriter()
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
