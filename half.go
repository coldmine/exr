package exr

import "math"

// half converts a uint16 to 32bit IEEE float.
func half(h uint16) float32 {
	var x uint32

	if h&0x7FFF == 0 { // Signed zero
		x = uint32(h) << 16 // Return the signed zero
	} else { // Not zero
		hs := h & 0x8000 // Pick off sign bit
		he := h & 0x7C00 // Pick off exponent bits
		hm := h & 0x03FF // Pick off mantissa bits

		if he == 0 { // Denormal will convert to normalized
			e := 0 // The following loop figures out how much extra to adjust the exponent
			hm <<= 1

			for hm&0x0400 == 0 {
				e++
				hm <<= 1
			} // Shift until leading bit overflows into exponent bit
			xs := uint32(hs) << 16                // Sign bit
			xes := (int(he >> 10)) - 15 + 127 - e // Exponent unbias the halfp, then bias the single
			xe := uint32(xes << 23)               // Exponent
			xm := uint32(hm&0x03FF) << 13         // Mantissa
			x = (xs | xe | xm)                    // Combine sign bit, exponent bits, and mantissa bits

		} else if he == 0x7C00 { // Inf or NaN (all the exponent bits are set)
			if hm == 0 { // If mantissa is zero ...
				x = (uint32(hs) << 16) | 0x7F800000 // Signed Inf
			} else {
				x = uint32(0xFFC00000) // NaN, only 1st mantissa bit set
			}
		} else { // Normalized number
			xs := uint32(hs) << 16        // Sign bit
			xes := int(he>>10) - 15 + 127 // Exponent unbias the halfp, then bias the single
			xe := uint32(xes << 23)       // Exponent
			xm := uint32(hm) << 13        // Mantissa
			x = (xs | xe | xm)            // Combine sign bit, exponent bits, and mantissa bits
		}
	}

	return math.Float32frombits(x)
}
