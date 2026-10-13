package termimg

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestQuerySequenceAsksForGraphicsThenDA1(t *testing.T) {
	q := QuerySequence()
	if !strings.Contains(q, "a=q") || !strings.HasSuffix(q, "\x1b[c") {
		t.Fatalf("query = %q", q)
	}
}

func TestFitKeepsAspectAndBounds(t *testing.T) {
	cases := []struct{ w, h, mc, mr int }{{800, 600, 60, 20}, {100, 1000, 60, 20}, {4000, 100, 50, 10}}
	for _, c := range cases {
		cols, rows := Fit(c.w, c.h, c.mc, c.mr)
		if cols < 1 || rows < 1 || cols > c.mc || rows > c.mr {
			t.Errorf("Fit(%v) = %d,%d out of bounds", c, cols, rows)
		}
	}
	if cols, rows := Fit(0, 0, 10, 10); cols != 0 || rows != 0 {
		t.Fatal("empty image must fit nothing")
	}
}

func TestPlaceholdersShape(t *testing.T) {
	lines := Placeholders(7, 4, 3)
	if len(lines) != 3 {
		t.Fatalf("rows = %d", len(lines))
	}
	if n := strings.Count(lines[0], string(rune(0x10EEEE))); n != 4 {
		t.Fatalf("cols = %d", n)
	}
	if !strings.Contains(lines[0], "38;2;0;0;7m") {
		t.Fatalf("id must travel in the foreground colour: %q", lines[0])
	}
}

func writePNG(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		img.Set(x, 0, color.RGBA{255, 0, 0, 255})
	}
	p := filepath.Join(t.TempDir(), "i.png")
	f, _ := os.Create(p)
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEncodeBuildsVirtualPlacementTransmission(t *testing.T) {
	seq, err := Encode(writePNG(t, 64, 32), 8, 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"\x1b_G", "U=1", "c=8", "r=4", "f=100"} {
		if !strings.Contains(seq, want) {
			t.Errorf("sequence lacks %q: %.120q", want, seq)
		}
	}
	got := WithID(seq, 42)
	if !strings.Contains(got, "i=42") || strings.Contains(got, "i=16777215") {
		t.Fatalf("id not replaced: %.120q", got)
	}
}

func TestEncodeRejectsDecompressionBombs(t *testing.T) {
	// a header that declares 100000 x 100000 pixels must be refused before decoding
	p := writePNG(t, 8, 8)
	b, _ := os.ReadFile(p)
	// patch IHDR width/height (bytes 16..23); the CRC is not checked by DecodeConfig
	copy(b[16:20], []byte{0, 1, 0x86, 0xA0})
	copy(b[20:24], []byte{0, 1, 0x86, 0xA0})
	_ = os.WriteFile(p, b, 0o644)
	if _, err := Encode(p, 8, 4); err == nil {
		t.Fatal("want error for oversized image")
	}
}

func TestEncodeMissingFile(t *testing.T) {
	if _, err := Encode(filepath.Join(t.TempDir(), "nope.png"), 8, 4); err == nil {
		t.Fatal("want error")
	}
}

// Every format the preview accepts must also encode for the terminal (Mathias, 2026-10-13: "hay imagenes que no se
// pueden mirar"); the fixtures are the ones of the preview tests.
func TestEncodeEveryFormatOfThePreview(t *testing.T) {
	for _, name := range []string{"baseline.jpg", "progressive.jpg", "cmyk.jpg", "photo.png", "interlaced.gif", "photo.webp"} {
		seq, err := Encode(filepath.Join("..", "preview", "testdata", "images", name), 20, 8)
		if err != nil || seq == "" {
			t.Errorf("%s: %v", name, err)
		}
	}
}
