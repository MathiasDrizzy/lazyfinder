package preview

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const (
	maxArchiveEntries = 500    // lines shown
	maxArchiveScan    = 200000 // entries counted in a tar before giving up on the total
)

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

// clean makes an archive's own text safe for the screen: control characters become "?".
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 || (r >= 0x7f && r < 0xa0) {
			return '?'
		}
		return r
	}, s)
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
		lines = append(lines, fmt.Sprintf("%10s  %s", sz, clean(name)))
	}
	if kind == zipArchive {
		zr, err := zip.OpenReader(path)
		if err != nil {
			return nil, 0, false, err
		}
		defer func() { _ = zr.Close() }()
		for _, f := range zr.File {
			add(f.Name, int64(f.UncompressedSize64), f.FileInfo().IsDir())
		}
		return lines, total, false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, false, err
	}
	defer func() { _ = f.Close() }()
	var r io.Reader = f
	if kind == tarGzArchive {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, 0, false, err
		}
		defer func() { _ = gz.Close() }()
		r = gz
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
		if err != nil {
			if total == 0 {
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
