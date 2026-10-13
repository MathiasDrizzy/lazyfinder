package preview

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mathias, 2026-10-13: "hay imagenes que no se pueden mirar". A JPEG from an AI tool or a camera carries a C2PA
// manifest in APP11 (JUMBF) segments before the picture data; they can be far bigger than the 8 KiB the sniffing read,
// so the picture was taken for a binary file and shown as hex.
func jpegWithApp11(t *testing.T, size int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for x := 0; x < 40; x++ {
		for y := 0; y < 30; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 6), uint8(y * 8), 90, 255})
		}
	}
	var base bytes.Buffer
	if err := jpeg.Encode(&base, img, nil); err != nil {
		t.Fatal(err)
	}
	raw := base.Bytes()
	var out bytes.Buffer
	out.Write(raw[:2])            // SOI
	for left := size; left > 0; { // APP11 (0xFFEB) segments of "JP" + box header + payload
		n := min(left, 60000)
		seg := append([]byte("JP\x00\x01\x00\x00\x00\x00jumb\x00\x00\x00\x1ajumdc2pa"), bytes.Repeat([]byte{0}, n)...)
		out.Write([]byte{0xFF, 0xEB, byte((len(seg) + 2) >> 8), byte(len(seg) + 2)})
		out.Write(seg)
		left -= n
	}
	out.Write(raw[2:])
	p := filepath.Join(t.TempDir(), "Gemini_Generated_Image_abc.jpg")
	if err := os.WriteFile(p, out.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAJPEGWithABigC2PAManifestIsAnImageNotHex(t *testing.T) {
	for _, size := range []int{500, 20000, 150000} {
		r := load(t, jpegWithApp11(t, size), 1<<20)
		if r.Kind != KindImage || r.ImageW != 40 || r.ImageH != 30 {
			t.Errorf("APP11 of %d bytes: kind %v, %dx%d", size, r.Kind, r.ImageW, r.ImageH)
		}
	}
}

func TestEveryImageFormatIsAnImage(t *testing.T) {
	for _, name := range []string{"baseline.jpg", "progressive.jpg", "cmyk.jpg", "photo.png", "interlaced.gif", "photo.webp"} {
		r := load(t, filepath.Join("testdata", "images", name), 1<<20)
		if r.Kind != KindImage || r.ImageW == 0 || r.ImageH == 0 {
			t.Errorf("%s: kind %v, %dx%d, err %v", name, r.Kind, r.ImageW, r.ImageH, r.Err)
		}
	}
}

// A picture that cannot be drawn (a format with no decoder, a file over the size limit) is described in words, never
// as a hex dump.
func TestAnImageThatCannotBeDrawnIsDescribedNotDumped(t *testing.T) {
	dir := t.TempDir()
	heic := filepath.Join(dir, "IMG_0001.HEIC")
	if err := os.WriteFile(heic, append([]byte("\x00\x00\x00\x18ftypheic\x00\x00\x00\x00mif1heic"), make([]byte, 4096)...), 0o600); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "huge.png")
	src, err := os.ReadFile(filepath.Join("testdata", "images", "photo.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(big, append(src, make([]byte, imageMaxBytes+10)...), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{heic, big} {
		r := Load(context.Background(), p, Options{MaxTextBytes: 1 << 20, Width: 80})
		if r.Kind == KindBinary {
			t.Errorf("%s was dumped as hex:\n%s", filepath.Base(p), plain(r))
		}
		if txt := plain(r); !strings.Contains(txt, "Image") && !strings.Contains(txt, "Imagen") {
			t.Errorf("%s: no description of the image:\n%s", filepath.Base(p), txt)
		}
	}
}
