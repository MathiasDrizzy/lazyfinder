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
	_ "golang.org/x/image/bmp"  // register decoders for DecodeConfig
	_ "golang.org/x/image/tiff" // register decoders for DecodeConfig
	_ "golang.org/x/image/webp" // register decoders for DecodeConfig
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
	f, _, err := fsmodel.OpenRegular(path)
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
	// The header is read from the file itself, not from the sniffed bytes: a JPEG from a camera or an AI tool carries
	// C2PA/JUMBF segments before the picture that can be bigger than the sniff.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return failed(res, path, err)
	}
	cfg, _, cfgErr := image.DecodeConfig(f)
	if _, err := f.Seek(int64(len(head)), io.SeekStart); err != nil {
		return failed(res, path, err)
	}
	switch {
	case cfgErr == nil && info.Size() <= imageMaxBytes:
		res.Kind, res.ImageW, res.ImageH = KindImage, cfg.Width, cfg.Height
		res.Facts = append(res.Facts, fmt.Sprintf(i18n.T("Dimensiones: %d x %d", "Dimensions: %d x %d"), cfg.Width, cfg.Height))
		return res
	case cfgErr == nil:
		return describeImage(res, info, cfg.Width, cfg.Height, i18n.T("pesa demasiado para mostrarla aquí", "too big to draw here"))
	case isImageByName(filepath.Base(path)):
		return describeImage(res, info, 0, 0, i18n.T("este formato no se puede mostrar aquí", "this format cannot be drawn here"))
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
	body = []byte(Visible(string(body)))
	if ext := extOf(name); (ext == "md" || ext == "markdown") && !opts.MarkdownRaw && !truncated {
		if lines, ok := renderMarkdown(ctx, string(body), opts.Width, opts.NoColor); ok {
			res.Lines = lines
			return res
		}
	}
	if sep, ok := tableSeparators[extOf(name)]; ok {
		if lines, ok := renderTable(wholeRows(string(body), truncated), sep, opts.NoColor); ok {
			res.Lines = lines
			return res
		}
	}
	res.Lines = highlight(name, body, opts.NoColor)
	return res
}

// imageExts are the extensions of pictures; one that cannot be drawn is described, never dumped as hex.
var imageExts = map[string]bool{
	"jpg": true, "jpeg": true, "png": true, "gif": true, "webp": true, "bmp": true, "tif": true, "tiff": true,
	"heic": true, "heif": true, "avif": true, "jxl": true, "ico": true, "icns": true, "psd": true,
	"cr2": true, "cr3": true, "nef": true, "arw": true, "dng": true, "raf": true, "orf": true, "rw2": true,
}

func isImageByName(name string) bool { return imageExts[extOf(name)] }

// describeImage is what the popup shows for a picture it cannot draw: its facts in words.
func describeImage(res Result, info os.FileInfo, w, h int, why string) Result {
	res.Kind = KindText
	kind := strings.ToUpper(extOf(info.Name()))
	lines := []string{fmt.Sprintf(i18n.T("Imagen %s", "Image %s"), kind)}
	if w > 0 && h > 0 {
		lines = append(lines, fmt.Sprintf(i18n.T("Dimensiones: %d x %d", "Dimensions: %d x %d"), w, h))
	}
	lines = append(lines,
		fmt.Sprintf(i18n.T("Tamaño: %s", "Size: %s"), fsmodel.HumanSize(info.Size())),
		fmt.Sprintf(i18n.T("Modificado: %s", "Modified: %s"), fsmodel.FormatDate(info.ModTime(), time.Now())),
		"", why+i18n.T(". Pulsa Q para Vista rápida.", ". Press Q for Quick Look."))
	res.Lines = lines
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
			Discard(png)
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
		name := VisibleLine(e.Name)
		if e.IsDir {
			name += "/"
		}
		res.Lines = append(res.Lines, name)
	}
	return res
}

// highlightBudget is how long a preview may spend coloring a file before it shows the text plain.
const highlightBudget = 500 * time.Millisecond

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
	it, err := lexer.Tokenise(nil, text) // not chroma.Coalesce: it reads ahead, past any budget
	if err != nil {
		return strings.Split(text, "\n")
	}
	// Some lexers are quadratic on hostile input (512 KB of [ in a .cpp file took half an hour), and chroma takes no
	// context: the tokens are pulled one by one, and past highlightBudget the text shows plain.
	deadline := time.Now().Add(highlightBudget)
	var tokens []chroma.Token
	var run strings.Builder // the value of the last token, which grows while the next ones have its type
	for t := it(); t != chroma.EOF; t = it() {
		if time.Now().After(deadline) {
			return strings.Split(text, "\n")
		}
		if n := len(tokens); n > 0 && tokens[n-1].Type == t.Type { // what Coalesce did: fewer color sequences
			run.WriteString(t.Value)
			continue
		}
		if n := len(tokens); n > 0 {
			tokens[n-1].Value = run.String()
		}
		run.Reset()
		run.WriteString(t.Value)
		tokens = append(tokens, t)
	}
	if n := len(tokens); n > 0 {
		tokens[n-1].Value = run.String()
	}
	var b strings.Builder
	if err := formatters.TTY256.Format(&b, styles.Get(highlightStyle), chroma.Literator(tokens...)); err != nil {
		return strings.Split(text, "\n")
	}
	return strings.Split(b.String(), "\n")
}
