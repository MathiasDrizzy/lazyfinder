package preview

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const maxArchiveEntries = 500 // lines shown

var (
	// maxArchiveScan is how many entries are counted in a tar before giving up on the total, and the most a zip may
	// have to be loaded at all: zip.NewReader keeps its whole central directory in memory.
	maxArchiveScan = 200000
	// maxArchiveBytes is how much of a compressed tar is inflated: skipping an entry means inflating it, and a small
	// archive may hold gigabytes.
	maxArchiveBytes int64 = 512 << 20
)

// errArchiveBudget stops the scan of a compressed tar once maxArchiveBytes are inflated.
var errArchiveBudget = errors.New("archive scan budget spent")

// budgetReader reads from r until left bytes are read or ctx is done, so a scan can always be stopped.
type budgetReader struct {
	ctx  context.Context
	r    io.Reader
	left int64
}

func (b *budgetReader) Read(p []byte) (int, error) {
	if err := b.ctx.Err(); err != nil {
		return 0, err
	}
	if b.left <= 0 {
		return 0, errArchiveBudget
	}
	n, err := b.r.Read(p[:min(int64(len(p)), b.left)])
	b.left -= int64(n)
	return n, err
}

func tooManyToList() string {
	return i18n.T("Demasiadas entradas para listarlas", "Too many entries to list")
}

// zipEntries reads the number of entries from the end record of a zip, or from its zip64 record, without loading the
// central directory. ok is false when there is no such record; zip.NewReader then says what is wrong.
func zipEntries(r io.ReaderAt, size int64) (n uint64, ok bool) {
	const eocdLen, maxComment, locatorLen, zip64Len = 22, 65535, 20, 56
	tail := min(size, eocdLen+maxComment)
	buf := make([]byte, tail)
	if _, err := r.ReadAt(buf, size-tail); err != nil && err != io.EOF {
		return 0, false
	}
	i := bytes.LastIndex(buf, []byte("PK\x05\x06"))
	if i < 0 || len(buf)-i < eocdLen {
		return 0, false
	}
	if n = uint64(binary.LittleEndian.Uint16(buf[i+10:])); n != 0xffff {
		return n, true
	}
	var loc [locatorLen]byte // a zip64 locator sits right before the end record
	at := size - tail + int64(i) - locatorLen
	if at < 0 {
		return 0, false
	}
	if _, err := r.ReadAt(loc[:], at); err != nil || string(loc[:4]) != "PK\x06\x07" {
		return 0, false
	}
	off := int64(binary.LittleEndian.Uint64(loc[8:]))
	var rec [zip64Len]byte
	if off < 0 || off > size-zip64Len {
		return 0, false
	}
	if _, err := r.ReadAt(rec[:], off); err != nil || string(rec[:4]) != "PK\x06\x06" {
		return 0, false
	}
	return binary.LittleEndian.Uint64(rec[32:]), true
}

type archiveKind int

const (
	notArchive archiveKind = iota
	zipArchive
	tarArchive
	tarGzArchive
)

func archiveOf(name string) archiveKind {
	n := strings.ToLower(name)
	switch {
	case strings.HasSuffix(n, ".zip"), strings.HasSuffix(n, ".jar"), strings.HasSuffix(n, ".docx"), strings.HasSuffix(n, ".xlsx"), strings.HasSuffix(n, ".pptx"):
		return zipArchive
	case strings.HasSuffix(n, ".tar.gz"), strings.HasSuffix(n, ".tgz"):
		return tarGzArchive
	case strings.HasSuffix(n, ".tar"):
		return tarArchive
	}
	return notArchive
}

// listArchive reads the table of contents of an archive and never extracts anything. Total is the number of
// entries (exact unless more is true), Lines are the first maxArchiveEntries of them.
func listArchive(ctx context.Context, path string, kind archiveKind) (lines []string, total int, more bool, err error) {
	add := func(name string, size int64, isDir bool) {
		total++
		if len(lines) >= maxArchiveEntries {
			return
		}
		sz := fsmodel.HumanSize(size)
		if isDir {
			sz = "--"
		}
		lines = append(lines, fmt.Sprintf("%10s  %s", sz, VisibleLine(name)))
	}
	f, info, err := fsmodel.OpenRegular(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer func() { _ = f.Close() }()
	if kind == zipArchive {
		if n, ok := zipEntries(f, info.Size()); ok && n > uint64(maxArchiveScan) {
			return []string{tooManyToList()}, int(min(n, uint64(1<<62))), false, nil
		}
		zr, err := zip.NewReader(f, info.Size())
		if err != nil {
			return nil, 0, false, err
		}
		for _, f := range zr.File {
			add(f.Name, int64(f.UncompressedSize64), f.FileInfo().IsDir())
		}
		return lines, total, false, nil
	}
	var r io.Reader = f
	if kind == tarGzArchive {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, 0, false, err
		}
		defer func() { _ = gz.Close() }()
		r = &budgetReader{ctx: ctx, r: gz, left: maxArchiveBytes}
	}
	tr := tar.NewReader(r)
	for {
		if ctx.Err() != nil {
			return nil, 0, false, ctx.Err()
		}
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if errors.Is(err, errArchiveBudget) {
			return lines, total, true, nil
		}
		if err != nil {
			if ctx.Err() != nil || total == 0 {
				return nil, 0, false, err
			}
			break // a damaged tail: show what was read
		}
		add(h.Name, h.Size, h.Typeflag == tar.TypeDir)
		if total >= maxArchiveScan {
			return lines, total, true, nil
		}
	}
	return lines, total, false, nil
}

func entriesFact(total int, more bool) string {
	switch {
	case more:
		return fmt.Sprintf(i18n.T("más de %d entradas", "more than %d entries"), total)
	case total == 1:
		return i18n.T("1 entrada", "1 entry")
	}
	return fmt.Sprintf(i18n.T("%d entradas", "%d entries"), total)
}
