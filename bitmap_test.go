package exr

import (
	"reflect"
	"testing"
)

func TestBitmap(t *testing.T) {
	cases := []struct {
		n     int
		nbyte int
		data  []uint16
		unset []uint16
		has   []uint16
		want  bitmap
	}{
		{
			n:     16,
			nbyte: 2,
			data:  []uint16{0, 0, 2, 3, 3, 3, 5, 9, 15},
			unset: []uint16{0},
			has:   []uint16{2, 3, 5, 9, 15},
			want:  bitmap{0b00101100, 0b10000010},
		},
	}
	for i, c := range cases {
		got := newBitmap(c.n)
		if len(got) != c.nbyte {
			t.Fatalf("nbyte[%d]: initialzed with length %d, want %d", i, len(got), c.nbyte)
		}
		for _, d := range c.data {
			got.Set(d)
		}
		for _, d := range c.unset {
			got.Unset(d)
		}
		for _, d := range c.has {
			if !got.Has(d) {
				t.Fatalf("has[%d]: doesn't have %d, that is expected to have", i, d)
			}
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("want[%d]: got %d, want %d", i, got, c.want)
		}
	}
}
