package theme

import (
	"image/color"
	"math"
)

// Luminance is the WCAG relative luminance of c, from 0 (black) to 1 (white).
func Luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	lin := func(v uint32) float64 {
		x := float64(v) / 65535
		if x <= 0.03928 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// Contrast is the WCAG contrast ratio between a and b, from 1 to 21.
func Contrast(a, b color.Color) float64 {
	la, lb := Luminance(a), Luminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// Readable returns fg when it already reaches the ratio min against bg. Otherwise it moves fg toward black or white,
// whichever gives more contrast, by the smallest step that gets there, so the hue of the palette is kept.
func Readable(fg, bg color.Color, min float64) color.Color {
	if Contrast(fg, bg) >= min {
		return fg
	}
	var target color.Color = color.RGBA{A: 255}
	if Contrast(color.White, bg) > Contrast(color.Black, bg) {
		target = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	r0, g0, b0, _ := fg.RGBA()
	r1, g1, b1, _ := target.RGBA()
	mix := func(a, b uint32, t float64) uint8 {
		return uint8(math.Round((float64(a) + (float64(b)-float64(a))*t) / 257))
	}
	out := target
	for step := 1; step <= 100; step++ {
		t := float64(step) / 100
		c := color.RGBA{R: mix(r0, r1, t), G: mix(g0, g1, t), B: mix(b0, b1, t), A: 255}
		if Contrast(c, bg) >= min {
			out = c
			break
		}
	}
	return out
}
