// Package preview builds the content of the Space popup. Loading is a pure function of the file and the options,
// so the UI can run it in a goroutine and cancel it (docs/adr/0006-preview-and-images.md).
package preview

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/gif"  // register decoders for DecodeConfig
	_ "image/jpeg" // register decoders for DecodeConfig
	_ "image/png"  // register decoders for DecodeConfig
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/x/ansi"
)

// Kind classifies a preview result.
type Kind int

// Preview kinds.
const (
	KindText Kind = iota
	KindImage
	KindDir
	KindBinary
	KindOther // special files (pipes, devices, sockets): never read
	KindError
	KindCancelled
	KindArchive // a listing of what is inside, never extracted
	KindMedia   // facts about audio or video
)

const (
	sniffBytes     = 8 << 10
	maxDirEntries  = 200
	imageMaxBytes  = 25 << 20
	highlightStyle = "monokai"
)

// Options controls a load.
type Options struct {
	MaxTextBytes int64
	Width        int
	NoColor      bool
	HexBytes     int  // bytes of a binary file shown as hex; 0 means DefaultHexBytes
	MarkdownRaw  bool // show Markdown as source instead of rendering it

	// Probe and Raster are the optional external tools for media facts and PDF pages; nil turns each off.
	Probe  Probe
	Raster Raster
}

// Result is what the popup shows.
type Result struct {
	Path      string
	Kind      Kind
	Lines     []string // ANSI-styled lines for text and directories
	Facts     []string // "Kind: PNG image", "Size: 1.2 MB", ...
	Truncated bool
	ImageW    int
	ImageH    int
	ImagePath string // the file to draw when it is not Path itself (the first page of a PDF)
	Err       error
}

// StripANSI returns Lines without styling, for tests and plain output.
func (r Result) StripANSI() []string {
	out := make([]string, len(r.Lines))
	for i, l := range r.Lines {
		out[i] = ansi.Strip(l)
	}
	return out
}

// Load builds the preview of path. It never reads special files and reads at most opts.MaxTextBytes of text.
func Load(ctx context.Context, path string, opts Options) Result {
	res := Result{Path: path}
	if ctx.Err() != nil {
		res.Kind = KindCancelled
		return res
	}
	info, err := os.Stat(path)
	if err != nil {
		return failed(res, path, err)
	}
	res.Facts = facts(path, info)
	switch {
	case info.IsDir():
		res.Kind = KindDir
		return loadDir(ctx, res, path)
	case !info.Mode().IsRegular():
		res.Kind = KindOther
		return res
	}
	f, err := os.Open(path)
	if err != nil {
		return failed(res, path, err)
	}
	defer func() { _ = f.Close() }()

	head := make([]byte, sniffBytes)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return failed(res, path, err)
	}
	head = head[:n]
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(head)); err == nil && info.Size() <= imageMaxBytes {
		res.Kind, res.ImageW, res.ImageH = KindImage, cfg.Width, cfg.Height
		res.Facts = append(res.Facts, fmt.Sprintf(i18n.T("Dimensiones: %d x %d", "Dimensions: %d x %d"), cfg.Width, cfg.Height))
		return res
	}
	name := filepath.Base(path)
	if r, ok := loadSpecial(ctx, res, path, name, opts); ok {
		return r
	}
	if bytes.IndexByte(head, 0) >= 0 || isBinaryByName(name) {
		return loadHex(res, f, info.Size(), opts)
	}
	body, truncated, err := readLimited(f, head, opts.MaxTextBytes)
	if err != nil {
		return failed(res, path, err)
	}
	if ctx.Err() != nil {
		res.Kind = KindCancelled
		return res
	}
	res.Kind, res.Truncated = KindText, truncated
	if ext := extOf(name); (ext == "md" || ext == "markdown") && !opts.MarkdownRaw && !truncated {
		if lines, ok := renderMarkdown(string(body), opts.Width, opts.NoColor); ok {
			res.Lines = lines
			return res
		}
	}
	res.Lines = highlight(name, body, opts.NoColor)
	return res
}

// isBinaryByName says whether the name alone makes the file binary even if its first bytes look like text: a
// PDF or media file whose tool is missing gets the hex view, not a wall of garbage.
func isBinaryByName(name string) bool {
	e := extOf(name)
	return e == "pdf" || mediaExts[e] || archiveOf(name) != notArchive
}

// loadSpecial handles the formats that have their own view: archives, media and PDF. ok is false when the file
// is none of those, or the view could not be made (the caller then falls back to hex).
func loadSpecial(ctx context.Context, res Result, path, name string, opts Options) (Result, bool) {
	if k := archiveOf(name); k != notArchive {
		lines, total, more, err := listArchive(ctx, path, k)
		if err != nil {
			return res, false
		}
		res.Kind, res.Lines, res.Truncated = KindArchive, lines, total > len(lines)
		res.Facts = append(res.Facts, entriesFact(total, more))
		return res, true
	}
	switch ext := extOf(name); {
	case mediaExts[ext] && opts.Probe != nil:
		facts, err := opts.Probe(ctx, path)
		if err != nil || len(facts) == 0 {
			return res, false
		}
		res.Kind, res.Lines = KindMedia, facts
		return res, true
	case ext == "pdf" && opts.Raster != nil:
		png, err := opts.Raster(ctx, path)
		if err != nil {
			return res, false
		}
		pf, err := os.Open(png)
		if err != nil {
			return res, false
		}
		defer func() { _ = pf.Close() }()
		cfg, _, err := image.DecodeConfig(pf)
		if err != nil {
			Discard(png)
			return res, false
		}
		res.Kind, res.ImagePath, res.ImageW, res.ImageH = KindImage, png, cfg.Width, cfg.Height
		return res, true
	}
	return res, false
}

// loadHex shows the first bytes of a binary file as a hex dump.
func loadHex(res Result, f *os.File, size int64, opts Options) Result {
	n := opts.HexBytes
	if n <= 0 {
		n = DefaultHexBytes
	}
	buf := make([]byte, n)
	got, err := f.ReadAt(buf, 0)
	if err != nil && err != io.EOF {
		return failed(res, res.Path, err)
	}
	res.Kind, res.Lines, res.Truncated = KindBinary, hexRows(buf[:got]), size > int64(got)
	return res
}

func failed(res Result, path string, err error) Result {
	res.Kind = KindError
	res.Err = fmt.Errorf("cannot preview %q: %w", filepath.Base(path), unwrapPath(err))
	return res
}

func unwrapPath(err error) error {
	if pe, ok := err.(*os.PathError); ok {
		return pe.Err
	}
	return err
}

// readLimited continues reading f after head until max bytes in total.
func readLimited(f *os.File, head []byte, max int64) (body []byte, truncated bool, err error) {
	if int64(len(head)) >= max {
		return head[:max], true, nil
	}
	rest, err := io.ReadAll(io.LimitReader(f, max-int64(len(head))+1))
	if err != nil {
		return nil, false, err
	}
	body = append(head, rest...)
	if int64(len(body)) > max {
		return body[:max], true, nil
	}
	return body, false, nil
}

func facts(path string, info os.FileInfo) []string {
	isLink := false
	if li, err := os.Lstat(path); err == nil {
		isLink = li.Mode()&os.ModeSymlink != 0
	}
	f := []string{
		i18n.T("Clase: ", "Kind: ") + fsmodel.KindOf(filepath.Base(path), info.IsDir(), isLink),
	}
	if !info.IsDir() {
		f = append(f, i18n.T("Tamaño: ", "Size: ")+fsmodel.HumanSize(info.Size()))
	}
	return append(f, i18n.T("Modificado: ", "Modified: ")+fsmodel.FormatDate(info.ModTime(), time.Now()))
}

func loadDir(ctx context.Context, res Result, path string) Result {
	l, err := fsmodel.ReadDir(path, fsmodel.ReadOptions{ShowHidden: true})
	if err != nil {
		res.Kind, res.Err = KindError, err
		return res
	}
	res.Facts = append(res.Facts, fmt.Sprintf("%d items", len(l.Entries)))
	for i, e := range l.Entries {
		if ctx.Err() != nil {
			res.Kind = KindCancelled
			return res
		}
		if i == maxDirEntries {
			res.Truncated = true
			break
		}
		name := e.Name
		if e.IsDir {
			name += "/"
		}
		res.Lines = append(res.Lines, name)
	}
	return res
}

func highlight(name string, body []byte, noColor bool) []string {
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if noColor {
		return strings.Split(text, "\n")
	}
	lexer := lexers.Match(name)
	if lexer == nil {
		lexer = lexers.Analyse(text)
	}
	if lexer == nil {
		return strings.Split(text, "\n")
	}
	lexer = chroma.Coalesce(lexer)
	it, err := lexer.Tokenise(nil, text)
	if err != nil {
		return strings.Split(text, "\n")
	}
	var b strings.Builder
	if err := formatters.TTY256.Format(&b, styles.Get(highlightStyle), it); err != nil {
		return strings.Split(text, "\n")
	}
	return strings.Split(b.String(), "\n")
}
