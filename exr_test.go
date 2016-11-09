package exr

import (
	"testing"
)

func TestDecode(t *testing.T) {
	// These are all valid exr files.
	cases := []string{
		"image/scanline.exr",
		"image/singlepart.exr",
		"image/multipart.exr",
	}

	for _, c := range cases {
		_, err := Decode(c)
		if err != nil {
			t.Fatal("Could not decode exr image: %v: %v", c, err)
		}
	}
}
