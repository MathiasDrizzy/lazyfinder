package preview

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const maxArchiveEntries = 500 // lines shown

var (
	// maxArchiveScan is how many entries are counted in an archive before giving up on the total.
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
		dir, ok := zipDirectory(f, info.Size())
		if !ok {
			return nil, 0, false, zip.ErrFormat
		}
		if dir.records > uint64(maxArchiveScan) {
			return []string{tooManyToList()}, int(min(dir.records, uint64(1<<62))), false, nil
		}
		stopped, err := walkZip(ctx, f, info.Size(), dir.start, func(name string, size int64, isDir bool) bool {
			add(name, size, isDir)
			return total <= maxArchiveScan
		})
		if err != nil {
			return nil, 0, false, err
		}
		if stopped {
			return []string{tooManyToList()}, total, true, nil
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
