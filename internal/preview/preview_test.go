package preview

import (
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func load(t *testing.T, path string, max int64) Result {
	t.Helper()
	return Load(context.Background(), path, Options{MaxTextBytes: max, Width: 80})
}

func write(t *testing.T, name, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func plain(r Result) string {
	return strings.Join(r.StripANSI(), "\n")
}

func TestTextIsHighlightedAndKeepsContent(t *testing.T) {
	p := write(t, "main.go", "package main\n\nfunc main() {}\n")
	r := load(t, p, 1<<20)
	if r.Kind != KindText {
		t.Fatalf("kind = %v", r.Kind)
	}
	if !strings.Contains(plain(r), "func main() {}") {
		t.Fatalf("content lost: %q", plain(r))
	}
	if !strings.Contains(strings.Join(r.Lines, "\n"), "\x1b[") {
		t.Fatal("expected ANSI highlighting")
	}
}

func TestTextTruncatedAtLimit(t *testing.T) {
	p := write(t, "big.txt", strings.Repeat("0123456789\n", 1000))
	r := load(t, p, 100)
	if !r.Truncated {
		t.Fatal("Truncated must be set")
	}
	if got := len(plain(r)); got > 400 {
		t.Fatalf("read too much: %d", got)
	}
}

func TestBinaryShowsFactsNotBytes(t *testing.T) {
	p := write(t, "blob.bin", "abc\x00\x01\x02def")
	r := load(t, p, 1<<20)
	if r.Kind != KindBinary {
		t.Fatalf("kind = %v", r.Kind)
	}
	if strings.Contains(plain(r), "\x00") {
		t.Fatal("binary bytes leaked")
	}
	if len(r.Facts) == 0 {
		t.Fatal("expected facts")
	}
}

func TestImageReportsDimensions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "p.png")
	f, _ := os.Create(p)
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 40, 20))); err != nil {
		t.Fatal(err)
	}
	f.Close()
	r := load(t, p, 1<<20)
	if r.Kind != KindImage || r.ImageW != 40 || r.ImageH != 20 {
		t.Fatalf("r = %+v", r)
	}
}

func TestDirectoryListsEntriesAndCount(t *testing.T) {
	d := t.TempDir()
	for _, n := range []string{"a.txt", "b.md"} {
		_ = os.WriteFile(filepath.Join(d, n), []byte("x"), 0o644)
	}
	_ = os.Mkdir(filepath.Join(d, "sub"), 0o755)
	r := load(t, d, 1<<20)
	if r.Kind != KindDir {
		t.Fatalf("kind = %v", r.Kind)
	}
	s := plain(r)
	for _, want := range []string{"sub", "a.txt", "b.md"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %q", want, s)
		}
	}
	if !strings.Contains(strings.Join(r.Facts, " "), "3 items") {
		t.Fatalf("facts = %v", r.Facts)
	}
}

func TestFIFOIsNeverRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), "pipe")
	if err := syscall.Mkfifo(p, 0o644); err != nil {
		t.Skip("mkfifo unavailable")
	}
	done := make(chan Result, 1)
	go func() { done <- load(t, p, 1<<20) }()
	select {
	case r := <-done:
		if r.Kind != KindOther {
			t.Fatalf("kind = %v", r.Kind)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Load blocked on a FIFO")
	}
}

func TestHugeSparseFileReadsOnlyTheLimit(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sparse.txt")
	f, _ := os.Create(p)
	_ = f.Truncate(1 << 30)
	f.Close()
	start := time.Now()
	r := load(t, p, 4096)
	if time.Since(start) > 2*time.Second {
		t.Fatal("too slow")
	}
	if r.Kind == KindError {
		t.Fatalf("err: %v", r.Err)
	}
}

func TestCancelledContextReturnsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p := write(t, "a.txt", "hello")
	r := Load(ctx, p, Options{MaxTextBytes: 100, Width: 80})
	if r.Kind != KindCancelled {
		t.Fatalf("kind = %v", r.Kind)
	}
}

func TestUnreadableFileIsAnActionableError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip()
	}
	p := write(t, "secret.txt", "x")
	_ = os.Chmod(p, 0)
	r := load(t, p, 100)
	if r.Kind != KindError || !strings.Contains(r.Err.Error(), "secret.txt") {
		t.Fatalf("r = %+v", r)
	}
}
