package preview

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadWith(t *testing.T, path string, o Options) Result {
	t.Helper()
	if o.MaxTextBytes == 0 {
		o.MaxTextBytes = 1 << 20
	}
	if o.Width == 0 {
		o.Width = 80
	}
	return Load(context.Background(), path, o)
}

func makeZip(t *testing.T, names map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "a.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for n, body := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return p
}

func makeTar(t *testing.T, name string, gz bool, names map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var tw *tar.Writer
	var gw *gzip.Writer
	if gz {
		gw = gzip.NewWriter(f)
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(f)
	}
	for n, body := range names {
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	if gw != nil {
		_ = gw.Close()
	}
	return p
}

func TestHexDumpOfABinaryFile(t *testing.T) {
	p := write(t, "blob.bin", "abc\x00\x01\x02def"+strings.Repeat("z", 20))
	r := loadWith(t, p, Options{})
	if r.Kind != KindBinary || len(r.Lines) != 2 {
		t.Fatalf("kind = %v, lines = %d", r.Kind, len(r.Lines))
	}
	want := "00000000  61 62 63 00 01 02 64 65  66 7a 7a 7a 7a 7a 7a 7a  |abc...defzzzzzzz|"
	if got := plain(r); !strings.HasPrefix(got, want) {
		t.Fatalf("first row:\n%q\nwant\n%q", strings.SplitN(got, "\n", 2)[0], want)
	}
	if !strings.HasPrefix(strings.Split(plain(r), "\n")[1], "00000010  ") {
		t.Fatal("second row starts at offset 0x10")
	}
}

func TestHexDumpIsCappedAndSaysSo(t *testing.T) {
	p := write(t, "big.bin", strings.Repeat("\x00\x01", 5000))
	r := loadWith(t, p, Options{HexBytes: 64})
	if len(r.Lines) != 4 || !r.Truncated {
		t.Fatalf("lines = %d truncated = %v, want 4 rows of 16 and truncated", len(r.Lines), r.Truncated)
	}
}

func TestZipIsListedNotExtracted(t *testing.T) {
	p := makeZip(t, map[string]string{"a.txt": "hello", "dir/b.go": "package b"})
	r := loadWith(t, p, Options{})
	if r.Kind != KindArchive {
		t.Fatalf("kind = %v", r.Kind)
	}
	got := plain(r)
	for _, want := range []string{"a.txt", "dir/b.go", "5 bytes", "9 bytes"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if !strings.Contains(strings.Join(r.Facts, " "), "2 entries") {
		t.Errorf("facts = %v", r.Facts)
	}
}

func TestTarAndTarGzAreListed(t *testing.T) {
	for _, tc := range []struct {
		name string
		gz   bool
	}{{"x.tar", false}, {"x.tar.gz", true}, {"x.tgz", true}} {
		p := makeTar(t, tc.name, tc.gz, map[string]string{"one.txt": "1", "two/three.txt": "333"})
		r := loadWith(t, p, Options{})
		if r.Kind != KindArchive || !strings.Contains(plain(r), "two/three.txt") {
			t.Errorf("%s: kind = %v\n%s", tc.name, r.Kind, plain(r))
		}
	}
}

func TestArchiveNamesCannotInjectTerminalCodes(t *testing.T) {
	p := makeZip(t, map[string]string{"evil\x1b[31m\x07name": "x"})
	r := loadWith(t, p, Options{})
	for _, l := range r.Lines {
		if strings.ContainsAny(l, "\x1b\x07") {
			t.Fatalf("control characters reached the screen: %q", l)
		}
	}
}

func TestBigArchiveListingIsCapped(t *testing.T) {
	names := map[string]string{}
	for i := 0; i < maxArchiveEntries+50; i++ {
		names[fmt.Sprintf("f%05d", i)] = "x"
	}
	r := loadWith(t, makeZip(t, names), Options{})
	if len(r.Lines) != maxArchiveEntries || !r.Truncated {
		t.Fatalf("lines = %d truncated = %v", len(r.Lines), r.Truncated)
	}
	if !strings.Contains(strings.Join(r.Facts, " "), fmt.Sprintf("%d entries", maxArchiveEntries+50)) {
		t.Errorf("facts must count all entries: %v", r.Facts)
	}
}

func TestACorruptArchiveFallsBackToHex(t *testing.T) {
	p := write(t, "broken.zip", "PK\x03\x04 this is not a zip\x00\x00")
	r := loadWith(t, p, Options{})
	if r.Kind != KindBinary || len(r.Lines) == 0 {
		t.Fatalf("kind = %v lines = %d", r.Kind, len(r.Lines))
	}
}

const mdSample = "# Title\n\nSome *emphasis* and a list:\n\n- first\n- second\n\n```go\nx := 1\n```\n"

func TestMarkdownIsRendered(t *testing.T) {
	r := loadWith(t, write(t, "r.md", mdSample), Options{})
	if r.Kind != KindText {
		t.Fatalf("kind = %v", r.Kind)
	}
	got := plain(r)
	for _, want := range []string{"Title", "emphasis", "first", "x := 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "```") || strings.Contains(got, "*emphasis*") {
		t.Errorf("markup must be rendered, not shown:\n%s", got)
	}
}

func TestMarkdownRawOptionShowsTheSource(t *testing.T) {
	r := loadWith(t, write(t, "r.md", mdSample), Options{MarkdownRaw: true})
	if !strings.Contains(plain(r), "```go") {
		t.Fatalf("raw mode shows the source:\n%s", plain(r))
	}
}

func TestMarkdownWithoutColorHasNoEscapes(t *testing.T) {
	r := loadWith(t, write(t, "r.md", mdSample), Options{NoColor: true})
	if strings.Contains(strings.Join(r.Lines, "\n"), "\x1b") {
		t.Fatal("NO_COLOR must produce no escape sequences")
	}
	if !strings.Contains(plain(r), "Title") {
		t.Fatal("content lost")
	}
}

func TestMediaFactsComeFromTheProbe(t *testing.T) {
	p := write(t, "clip.mp4", "\x00\x00\x00\x18ftypmp42")
	probe := func(_ context.Context, path string) ([]string, error) {
		if path != p {
			t.Errorf("probe got %q", path)
		}
		return []string{"Duration: 2:05", "Dimensions: 1920 x 1080"}, nil
	}
	r := loadWith(t, p, Options{Probe: probe})
	if r.Kind != KindMedia {
		t.Fatalf("kind = %v", r.Kind)
	}
	if got := plain(r); !strings.Contains(got, "Duration: 2:05") || !strings.Contains(got, "1920 x 1080") {
		t.Fatalf("lines:\n%s", got)
	}
}

func TestMediaWithoutAProbeOrWithAFailingOneFallsBackToHex(t *testing.T) {
	p := write(t, "clip.mov", "\x00\x00\x00\x18ftypqt  ")
	if r := loadWith(t, p, Options{}); r.Kind != KindBinary {
		t.Errorf("no probe: kind = %v", r.Kind)
	}
	fail := func(context.Context, string) ([]string, error) { return nil, errors.New("no tool") }
	if r := loadWith(t, p, Options{Probe: fail}); r.Kind != KindBinary {
		t.Errorf("failing probe: kind = %v", r.Kind)
	}
}

func TestParseMdlsKeepsTheUsefulFields(t *testing.T) {
	out := "kMDItemCodecs          = (\n    \"H.264\",\n    \"AAC\"\n)\nkMDItemDurationSeconds = 125.5\nkMDItemPixelHeight     = 1080\nkMDItemPixelWidth      = 1920\nkMDItemTitle           = (null)\n"
	got := strings.Join(ParseMdls(out), " | ")
	for _, want := range []string{"Duration: 2:05", "Dimensions: 1920 x 1080", "Codecs: H.264, AAC"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "null") {
		t.Errorf("empty fields must be dropped: %q", got)
	}
}

func fakeRaster(t *testing.T) func(context.Context, string) (string, error) {
	return func(_ context.Context, _ string) (string, error) {
		p := filepath.Join(t.TempDir(), "page.png")
		f, err := os.Create(p)
		if err != nil {
			return "", err
		}
		defer func() { _ = f.Close() }()
		return p, png.Encode(f, image.NewRGBA(image.Rect(0, 0, 30, 40)))
	}
}

func TestPDFShowsItsFirstPageAsAnImage(t *testing.T) {
	p := write(t, "doc.pdf", "%PDF-1.4\n%\xe2\xe3\x00")
	r := loadWith(t, p, Options{Raster: fakeRaster(t)})
	if r.Kind != KindImage || r.ImageW != 30 || r.ImageH != 40 {
		t.Fatalf("kind = %v %dx%d", r.Kind, r.ImageW, r.ImageH)
	}
	if r.ImagePath == "" || r.ImagePath == p || r.Path != p {
		t.Fatalf("the page image is separate from the file: path=%q image=%q", r.Path, r.ImagePath)
	}
}

func TestPDFWithoutARasterizerOrWithAFailingOneFallsBackToHex(t *testing.T) {
	p := write(t, "doc.pdf", "%PDF-1.4 plain ascii only")
	if r := loadWith(t, p, Options{}); r.Kind != KindBinary {
		t.Errorf("no rasterizer: kind = %v", r.Kind)
	}
	fail := func(context.Context, string) (string, error) { return "", errors.New("no tool") }
	if r := loadWith(t, p, Options{Raster: fail}); r.Kind != KindBinary {
		t.Errorf("failing rasterizer: kind = %v", r.Kind)
	}
}
