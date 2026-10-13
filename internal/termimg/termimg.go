// Package termimg shows images with the Kitty graphics protocol using Unicode placeholders: the image is
// transmitted once and the screen holds ordinary text cells (U+10EEEE with the id in the colour), so the pictures
// travel through Bubble Tea's renderer, popup compositing and multiplexers without ghost images.
// Spec: https://sw.kovidgoyal.net/kitty/graphics-protocol/ (docs/adr/0006-preview-and-images.md).
package termimg

import (
	"fmt"
	"image"
	_ "image/gif"  // decoders
	_ "image/jpeg" // decoders
	_ "image/png"  // decoders
	"os"
	"strconv"
	"strings"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/kitty"
)

const (
	queryID    = 31
	maxSide    = 1024
	cellAspect = 2.0
	pxPerCol   = 8
	// MaxPixels refuses images that would need too much memory to decode.
	MaxPixels = 50_000_000
	// MaxFileBytes is the largest image file lazyfinder reads.
	MaxFileBytes = 25 << 20
	// sentinelID is replaced by WithID when the sequence is sent.
	sentinelID = 16777215
)

// QuerySequence asks whether the terminal supports graphics (a=q) and requests DA1 right behind it: a graphics
// reply means support, a lone DA1 reply means none.
func QuerySequence() string {
	return ansi.KittyGraphics([]byte("AAAA"),
		"i="+strconv.Itoa(queryID), "s=1", "v=1", "a=q", "t=d", "f=24") + ansi.RequestPrimaryDeviceAttributes
}

// IsSupportReply reports whether e is the positive answer to QuerySequence.
func IsSupportReply(e uv.KittyGraphicsEvent) bool {
	return e.Options.ID == queryID && strings.HasPrefix(string(e.Payload), "OK")
}

// DeleteAllSequence deletes every visible image and frees its data, silently.
func DeleteAllSequence() string { return ansi.KittyGraphics(nil, "a=d", "d=A", "q=2") }

// Fit returns the cells an imgW x imgH picture needs inside maxCols x maxRows, keeping its proportions and never
// enlarging it past its natural size.
func Fit(imgW, imgH, maxCols, maxRows int) (cols, rows int) {
	if imgW <= 0 || imgH <= 0 || maxCols < 1 || maxRows < 1 {
		return 0, 0
	}
	cols = min(maxCols, max(2, imgW/pxPerCol))
	rows = int(float64(cols)*float64(imgH)/float64(imgW)/cellAspect + 0.5)
	if rows > maxRows {
		rows = maxRows
		cols = int(float64(rows)*cellAspect*float64(imgW)/float64(imgH) + 0.5)
	}
	return max(1, min(cols, maxCols)), max(1, rows)
}

// Placeholders returns the rows lines of cols cells that display image id.
func Placeholders(id, cols, rows int) []string {
	fg := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", id>>16&0xff, id>>8&0xff, id&0xff)
	lines := make([]string, rows)
	for r := range lines {
		var b strings.Builder
		b.WriteString(fg)
		for c := 0; c < cols; c++ {
			b.WriteRune(kitty.Placeholder)
			b.WriteRune(kitty.Diacritic(r))
			b.WriteRune(kitty.Diacritic(c))
		}
		b.WriteString("\x1b[0m")
		lines[r] = b.String()
	}
	return lines
}

// Encode decodes the image at path, shrinks it and builds its transmission sequence for cols x rows cells. It
// touches no shared state, so it is safe to run in a goroutine. Use WithID to give the sequence its real id.
func Encode(path string, cols, rows int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("cannot open image %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() || fi.Size() > MaxFileBytes {
		return "", fmt.Errorf("image %q is not a regular file under %d MiB", path, MaxFileBytes>>20)
	}
	// header first: a small file that declares billions of pixels is never decoded
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return "", fmt.Errorf("cannot read image header: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > MaxPixels {
		return "", fmt.Errorf("image is too large (%dx%d, maximum %d pixels)", cfg.Width, cfg.Height, MaxPixels)
	}
	if _, err := f.Seek(0, 0); err != nil {
		return "", err
	}
	img, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("cannot decode image: %w", err)
	}
	var b strings.Builder
	err = kitty.EncodeGraphics(&b, scaleDown(img, maxSide), &kitty.Options{
		Action: kitty.TransmitAndPut, ID: sentinelID, Format: kitty.PNG, Transmission: kitty.Direct,
		Chunk: true, Columns: cols, Rows: rows, VirtualPlacement: true, Quiet: 2,
	})
	return b.String(), err
}

// WithID puts the real id into a sequence made by Encode (later chunks only carry m and q).
func WithID(seq string, id int) string {
	end := strings.IndexByte(seq, ';')
	if end < 0 {
		return seq
	}
	return strings.Replace(seq[:end], "i="+strconv.Itoa(sentinelID), "i="+strconv.Itoa(id), 1) + seq[end:]
}

// scaleDown shrinks src so its longest side is at most limit, averaging each block of pixels.
func scaleDown(src image.Image, limit int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= limit && h <= limit {
		return src
	}
	nw, nh := max(1, w*limit/max(w, h)), max(1, h*limit/max(w, h))
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		y0, y1 := b.Min.Y+y*h/nh, b.Min.Y+max(y*h/nh+1, (y+1)*h/nh)
		for x := 0; x < nw; x++ {
			x0, x1 := b.Min.X+x*w/nw, b.Min.X+max(x*w/nw+1, (x+1)*w/nw)
			var r, g, bl, a, n uint32
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					pr, pg, pb, pa := src.At(sx, sy).RGBA()
					r, g, bl, a, n = r+pr, g+pg, bl+pb, a+pa, n+1
				}
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2], dst.Pix[i+3] = uint8(r/n>>8), uint8(g/n>>8), uint8(bl/n>>8), uint8(a/n>>8)
		}
	}
	return dst
}
