package exr

import (
	"reflect"
	"testing"
)

func TestHuffmanBuildCanonicalCodes(t *testing.T) {
	cases := []struct {
		pack []uint64
		want []uint64
	}{
		{
			pack: []uint64{
				3,
				4,
				4,
				3,
				2,
				4,
				4,
				2,
			},
			want: []uint64{
				0b010<<6 | 3,
				0b0000<<6 | 4,
				0b0001<<6 | 4,
				0b011<<6 | 3,
				0b10<<6 | 2,
				0b0010<<6 | 4,
				0b0011<<6 | 4,
				0b11<<6 | 2,
			},
		},
	}
	for _, c := range cases {
		huffmanBuildCanonicalCodes(c.pack)
		if !reflect.DeepEqual(c.pack, c.want) {
			t.Fatalf("want: %v, got: %v", c.want, c.pack)
		}
	}
}
