package preview

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	d, ok := zipDirectory(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if got := d.records; !ok || got != n {
		t.Fatalf("records = %d, %v; want %d", got, ok, n)
	}
}

// The end record and the central directory of a zip are untrusted bytes: any of them must give a listing or nothing,
// never a panic.
func FuzzZipDirectory(f *testing.F) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, _ = zw.Create("a")
	_ = zw.Close()
	f.Add(buf.Bytes())
	f.Add([]byte("PK\x05\x06"))
	f.Add(append(make([]byte, 20), []byte("PK\x06\x07PK\x05\x06\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff\xff")...))
	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		if d, ok := zipDirectory(r, int64(len(data))); ok {
			_, _ = walkZip(context.Background(), r, int64(len(data)), d.start, func(string, int64, bool) bool { return true })
		}
	})
}

// The count in the end record of a zip is 16 bits, and zip.NewReader checks it only modulo 65536: a zip of 65636
// entries whose end record says 100 got past the limit and was loaded whole (a zip of a few hundred MB that way cost
// gigabytes). The central directory is now read one record at a time and the scan stops at the limit.
func TestAZipThatLiesAboutItsCountIsNotLoaded(t *testing.T) {
	old := maxArchiveScan
	maxArchiveScan = 1000
	t.Cleanup(func() { maxArchiveScan = old })
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	const n = 1<<16 + 100
	for i := range n {
		if _, err := zw.CreateHeader(&zip.FileHeader{Name: fmt.Sprintf("%05d", i), Method: zip.Store}); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	end := bytes.LastIndex(b, []byte("PK\x05\x06"))
	binary.LittleEndian.PutUint16(b[end+8:], 100)  // records on this disk
	binary.LittleEndian.PutUint16(b[end+10:], 100) // records in all: n modulo 65536
	if zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b))); err != nil || len(zr.File) != n {
		t.Fatalf("the crafted zip must still open whole in archive/zip: %v", err)
	}
	p := filepath.Join(t.TempDir(), "liar.zip")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	lines, total, more, err := listArchive(context.Background(), p, zipArchive)
	if err != nil || !more || total <= maxArchiveScan || len(lines) != 1 || lines[0] != tooManyToList() {
		t.Fatalf("lines=%d total=%d more=%v err=%v, want the scan to stop at the limit", len(lines), total, more, err)
	}
}

// Reading the directory record by record lists what archive/zip lists: the names, the sizes (a zip64 one too) and
// which entries are folders, also behind data put before the zip.
func TestWalkZipListsWhatArchiveZipLists(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("#!/bin/sh\nexit 0\n") // a self-extracting stub
	zw := zip.NewWriter(&buf)
	zw.SetOffset(int64(buf.Len()))
	for _, h := range []*zip.FileHeader{{Name: "dir/"}, {Name: "dir/a.txt"}, {Name: "b.bin"}} {
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(h.Name, "/") {
			_, _ = w.Write([]byte(h.Name))
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	var want, got []string
	for _, f := range zr.File {
		want = append(want, fmt.Sprint(f.Name, f.UncompressedSize64, f.FileInfo().IsDir()))
	}
	d, ok := zipDirectory(bytes.NewReader(b), int64(len(b)))
	if !ok {
		t.Fatal("no directory")
	}
	if _, err := walkZip(context.Background(), bytes.NewReader(b), int64(len(b)), d.start, func(name string, size int64, isDir bool) bool {
		got = append(got, fmt.Sprint(name, size, isDir))
		return true
	}); err != nil || !slices.Equal(got, want) {
		t.Fatalf("walkZip = %q, %v; archive/zip = %q", got, err, want)
	}
	extra := binary.LittleEndian.AppendUint16(binary.LittleEndian.AppendUint16(nil, 1), 8)
	if zip64Size(binary.LittleEndian.AppendUint64(extra, 5<<32), 0xffffffff) != 5<<32 {
		t.Fatal("the zip64 size is not read")
	}
}
