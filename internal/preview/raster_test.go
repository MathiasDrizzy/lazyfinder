package preview

import (
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// hugePage writes a PDF of a few hundred bytes whose one page is 200 inches square, the largest the format allows.
func hugePage(t *testing.T) string {
	t.Helper()
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 14400 14400] >>",
	}
	b := []byte("%PDF-1.4\n")
	var offs []int
	for i, o := range objs {
		offs = append(offs, len(b))
		b = fmt.Appendf(b, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := len(b)
	b = fmt.Appendf(b, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, o := range offs {
		b = fmt.Appendf(b, "%010d 00000 n \n", o)
	}
	b = fmt.Appendf(b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	p := filepath.Join(t.TempDir(), "big.pdf")
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// pdftoppm drew page 1 at 110 dpi, and sips (the fallback then) at the page's own size, whatever that size: a
// 200-byte PDF with a 200-inch page took 2 GB and 10 s each time the cursor rested on it. The page comes out at most
// maxRasterSide pixels on its long side, whichever tool draws it.
func TestAHugePDFPageIsDrawnSmall(t *testing.T) {
	pdf := hugePage(t)
	for _, tool := range []string{"pdftoppm", "qlmanage"} {
		bin, err := exec.LookPath(tool)
		if err != nil {
			t.Logf("%s is not installed", tool)
			continue
		}
		t.Run(tool, func(t *testing.T) {
			only := t.TempDir() // PATH with just this tool, so the other one is not tried first
			if err := os.Symlink(bin, filepath.Join(only, tool)); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", only)
			page, err := SystemRaster(context.Background(), pdf)
			if err != nil {
				t.Fatal(err)
			}
			defer Discard(page)
			f, err := os.Open(page)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			cfg, err := png.DecodeConfig(f)
			if err != nil || max(cfg.Width, cfg.Height) > maxRasterSide {
				t.Fatalf("page %dx%d, %v", cfg.Width, cfg.Height, err)
			}
		})
	}
}
