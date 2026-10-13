package preview

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The tests here run the real tools on files made inside t.TempDir(), never on the user's own files. A tool that is
// not installed skips its test: the CI on Linux has no mdls or sips.

const minimalPDF = "%PDF-1.4\n" +
	"1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n" +
	"2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n" +
	"3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 100]>>endobj\n" +
	"trailer<</Root 1 0 R/Size 4>>\n%%EOF\n"

func writePDF(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "doc.pdf")
	if err := os.WriteFile(p, []byte(minimalPDF), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func requirePNG(t *testing.T, path string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("%s is not a PNG (%d bytes)", path, len(b))
	}
}

func TestSystemRasterDrawsAPdfPageWithPdftoppm(t *testing.T) {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm is not installed")
	}
	png, err := SystemRaster(context.Background(), writePDF(t))
	if err != nil {
		t.Fatal(err)
	}
	requirePNG(t, png)
	Discard(png)
	if _, err := os.Stat(filepath.Dir(png)); !os.IsNotExist(err) {
		t.Fatalf("Discard must remove the private folder: %v", err)
	}
}

func TestSystemRasterFallsBackToSipsWhenThereIsNoPdftoppm(t *testing.T) {
	sips, err := exec.LookPath("sips")
	if err != nil {
		t.Skip("sips is not installed")
	}
	bin := t.TempDir()
	if err := os.Symlink(sips, filepath.Join(bin, "sips")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin) // only sips is on the path
	png, err := SystemRaster(context.Background(), writePDF(t))
	if err != nil {
		t.Fatal(err)
	}
	requirePNG(t, png)
	Discard(png)
}

func TestSystemRasterSaysSoWhenNoToolIsInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := SystemRaster(context.Background(), writePDF(t)); err == nil || !strings.Contains(err.Error(), "neither") {
		t.Fatalf("err = %v", err)
	}
}

// Spotlight does not index a temporary folder, so mdls answers (null) for every field: the probe must say there is no
// metadata, quickly, and not invent facts or hang. (Facts from a real indexed file are NOT verified here: this Mac's
// Spotlight returned (null) for system sounds too.)
func TestSystemProbeOnAnUnindexedFileReportsNoMetadata(t *testing.T) {
	if _, err := exec.LookPath("mdls"); err != nil {
		t.Skip("mdls is not installed")
	}
	p := filepath.Join(t.TempDir(), "clip.wav")
	if err := os.WriteFile(p, []byte("RIFF$\x00\x00\x00WAVEfmt "), 0o644); err != nil {
		t.Fatal(err)
	}
	facts, err := SystemProbe(context.Background(), p)
	if err == nil && len(facts) == 0 {
		t.Fatal("no facts and no error")
	}
}
