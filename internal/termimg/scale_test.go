package termimg

import (
	"image"
	"image/color"
	"strings"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"
)

// A picture wider than maxSide is shrunk with its proportions kept and each block of pixels averaged.
func TestScaleDownKeepsTheProportionsAndAverages(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2*maxSide, maxSide))
	for y := range maxSide {
		for x := range 2 * maxSide {
			if x%2 == 0 {
				src.Set(x, y, color.RGBA{R: 255, A: 255})
			} else {
				src.Set(x, y, color.RGBA{B: 255, A: 255})
			}
		}
	}
	got := scaleDown(src, maxSide)
	if b := got.Bounds(); b.Dx() != maxSide || b.Dy() != maxSide/2 {
		t.Fatalf("size %v, want %dx%d", b, maxSide, maxSide/2)
	}
	r, _, bl, a := got.At(3, 3).RGBA()
	if r>>8 < 120 || r>>8 > 135 || bl>>8 < 120 || bl>>8 > 135 || a>>8 != 255 {
		t.Fatalf("pixel %d %d %d is not the average of red and blue", r>>8, bl>>8, a>>8)
	}
	small := image.NewRGBA(image.Rect(0, 0, 10, 10))
	if scaleDown(small, maxSide) != image.Image(small) {
		t.Fatal("a small picture must be kept as it is")
	}
}

func TestOnlyTheReplyToTheQueryMeansSupport(t *testing.T) {
	var ok, wrongID, refused uv.KittyGraphicsEvent
	ok.Options.ID, ok.Payload = queryID, []byte("OK")
	wrongID.Options.ID, wrongID.Payload = queryID+1, []byte("OK")
	refused.Options.ID, refused.Payload = queryID, []byte("ENOTSUPPORTED")
	if !IsSupportReply(ok) || IsSupportReply(wrongID) || IsSupportReply(refused) {
		t.Fatal("only an OK to the query id means support")
	}
	if d := DeleteAllSequence(); !strings.Contains(d, "a=d") || !strings.Contains(d, "d=A") || !strings.Contains(d, "q=2") {
		t.Fatalf("delete sequence = %q", d)
	}
}
