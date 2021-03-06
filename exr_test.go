package exr

import (
	"testing"
)

func TestDecode(t *testing.T) {
	// These are all valid exr files.
	cases := []string{
		"image/scanline.exr",
	}

	for _, c := range cases {
		_, err := Decode(c)
		if err != nil {
			t.Fatalf("Could not decode exr image: %v: %v", c, err)
		}
	}
}
