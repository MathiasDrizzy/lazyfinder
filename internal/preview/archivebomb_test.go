package preview

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Skipping an entry of a tar.gz means inflating it, so a small archive holding one huge entry kept the preview
// inflating gigabytes, with no way to cancel it until the entry ended. The scan now stops after a byte budget.
func TestATarGzBombStopsAfterTheByteBudget(t *testing.T) {
	old := maxArchiveBytes
	maxArchiveBytes = 1 << 20
	t.Cleanup(func() { maxArchiveBytes = old })
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range []struct {
		name string
		size int64
	}{{"bomb.bin", 8 << 20}, {"after.txt", 1}} {
		if err := tw.WriteHeader(&tar.Header{Name: e.name, Mode: 0o644, Size: e.size, Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(make([]byte, e.size)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "bomb.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	_, total, more, err := listArchive(context.Background(), p, tarGzArchive)
	if err != nil || !more || total != 1 {
		t.Fatalf("total=%d more=%v err=%v, want the scan to stop after the first entry", total, more, err)
	}
}

// zip.NewReader loads the whole central directory, so a zip made of millions of tiny entries cost gigabytes of
// memory. The count in its end record is read first, and a zip past the limit is not loaded.
func TestAZipWithTooManyEntriesIsNotLoaded(t *testing.T) {
	old := maxArchiveScan
	maxArchiveScan = 100
	t.Cleanup(func() { maxArchiveScan = old })
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for i := range 300 {
		if _, err := zw.Create(fmt.Sprintf("f%03d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "many.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	lines, total, more, err := listArchive(context.Background(), p, zipArchive)
	if err != nil || more || total != 300 || len(lines) != 1 || lines[0] != tooManyToList() {
		t.Fatalf("lines=%q total=%d more=%v err=%v, want the count without loading", lines, total, more, err)
	}
}

// Past 65535 entries the count lives in the zip64 end record.
func TestTheEntriesOfAZip64AreCountedFromItsRecord(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	const n = 70000
	for i := range n {
		if _, err := zw.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("%05d", i), Method: zip.Store}); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	got, ok := zipEntries(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if !ok || got != n {
		t.Fatalf("zipEntries = %d, %v; want %d", got, ok, n)
	}
}

// The end record of a zip is read from untrusted bytes: any of them must give a count or nothing, never a panic.
func FuzzZipEntries(f *testing.F) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("a")
	_ = zw.Close()
	f.Add(buf.Bytes())
	f.Add([]byte("PK\x05\x06"))
	f.Add(append(make([]byte, 20), []byte("PK\x06\x07PK\x05\x06\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff")...))
	f.Fuzz(func(t *testing.T, data []byte) {
		zipEntries(bytes.NewReader(data), int64(len(data)))
	})
}
