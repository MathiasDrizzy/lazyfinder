package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
	"github.com/MathiasDrizzy/lazyfinder/internal/preview"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/charmbracelet/x/ansi"
)

type imgState struct {
	id, cols, rows int
}

type previewState struct {
	open   bool
	id     int
	cancel context.CancelFunc
	res    *preview.Result
	scroll int
	img    *imgState
}

type menuState struct {
	open  bool
	path  string
	items []opener.Command
	sel   int
}

type previewMsg struct {
	id  int
	res preview.Result
}

type imageMsg struct {
	id         int
	seq        string
	cols, rows int
	err        error
}

func boxWidth(box []string) int {
	if len(box) == 0 {
		return 0
	}
	return ansi.StringWidth(box[0])
}

// previewGeom returns the popup rectangle; inner text is iw wide and bodyH tall (two rows go to facts and hints).
func (m *Model) previewGeom() (x, y, pw, ph, iw, bodyH int) {
	pw = min(m.w-6, max(50, m.w*3/4))
	ph = min(m.h-4, max(12, m.h*3/4))
	x, y = (m.w-pw)/2, (m.h-ph)/2
	iw, bodyH = pw-4, ph-2-2
	return
}

func (m *Model) openPreview() tea.Cmd {
	if _, ok := m.selected(); !ok {
		return nil
	}
	m.pv.open = true
	return m.refreshPreview()
}

func (m *Model) closePreview() tea.Cmd {
	hadImage := m.pv.img != nil
	if m.pv.cancel != nil {
		m.pv.cancel()
	}
	m.pv = previewState{id: m.pv.id}
	if hadImage {
		return tea.Raw(termimg.DeleteAllSequence())
	}
	return nil
}

// ImageShown says whether a preview image is still drawn in the terminal.
func (m *Model) ImageShown() bool { return m.pv.img != nil }

// quit leaves lazyfinder. An image of the preview is deleted first: it lives in the terminal, not in the frame, and
// would stay drawn over the shell.
func (m *Model) quit() tea.Cmd {
	return tea.Sequence(m.closePreview(), tea.Quit)
}

// handOver gives the terminal to a program (an editor, lazygit) once the preview and its image are gone.
func (m *Model) handOver(cmd *exec.Cmd) tea.Cmd {
	return tea.Sequence(m.closePreview(), tea.ExecProcess(cmd, func(err error) tea.Msg { return execDoneMsg{err: err} }))
}

// refreshPreview (re)starts loading the preview of the selection; stale loads are cancelled and their results ignored.
func (m *Model) refreshPreview() tea.Cmd {
	if !m.pv.open {
		return nil
	}
	if m.pv.cancel != nil {
		m.pv.cancel()
	}
	var drop tea.Cmd
	if m.pv.img != nil {
		drop = tea.Raw(termimg.DeleteAllSequence())
	}
	m.pv.id++
	m.pv.res, m.pv.img, m.pv.scroll = nil, nil, 0
	e, ok := m.selected()
	if !ok {
		return drop
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.pv.cancel = cancel
	id := m.pv.id
	_, _, _, _, iw, _ := m.previewGeom()
	pc := m.deps.Cfg.Preview
	opts := preview.Options{MaxTextBytes: pc.MaxTextBytes, Width: iw, NoColor: m.deps.NoColor, HexBytes: pc.HexBytes, MarkdownRaw: pc.Markdown == "raw"}
	if pc.ExternalTools != "off" {
		opts.Probe, opts.Raster = m.deps.Probe, m.deps.Raster
	}
	load := func() tea.Msg { return previewMsg{id: id, res: preview.Load(ctx, e.Path, opts)} }
	return tea.Batch(drop, load)
}

func (m *Model) onPreview(msg previewMsg) tea.Cmd {
	if !m.pv.open || msg.id != m.pv.id {
		return nil
	}
	res := msg.res
	m.pv.res = &res
	if res.Kind == preview.KindError {
		m.setError(res.Err)
	}
	if res.Kind != preview.KindImage || !m.imagesAllowed || m.deps.Cfg.Preview.Images == "off" {
		return nil
	}
	_, _, _, _, iw, bodyH := m.previewGeom()
	cols, rows := termimg.Fit(res.ImageW, res.ImageH, iw, bodyH)
	id, path := msg.id, res.Path
	if res.ImagePath != "" {
		path = res.ImagePath
	}
	return func() tea.Msg {
		defer preview.Discard(path) // only a page image of ours is removed
		seq, err := termimg.Encode(path, cols, rows)
		return imageMsg{id: id, seq: seq, cols: cols, rows: rows, err: err}
	}
}

func (m *Model) onImage(msg imageMsg) tea.Cmd {
	if !m.pv.open || msg.id != m.pv.id {
		return nil
	}
	if msg.err != nil {
		m.setError(msg.err)
		return nil
	}
	imgID := m.nextImgID
	m.nextImgID++
	m.pv.img = &imgState{id: imgID, cols: msg.cols, rows: msg.rows}
	return tea.Raw(termimg.WithID(msg.seq, imgID))
}

func (m *Model) previewKey(key string, act keymap.Action) tea.Cmd {
	switch {
	case act == keymap.Quit:
		return m.quit()
	case key == "esc" || act == keymap.Preview:
		return m.closePreview()
	case act == keymap.Up:
		return m.setSelection(m.sel - 1)
	case act == keymap.Down:
		return m.setSelection(m.sel + 1)
	case act == keymap.PageUp:
		m.scrollPreview(-m.pageSize())
	case act == keymap.PageDown:
		m.scrollPreview(m.pageSize())
	case key == "enter":
		return tea.Sequence(m.closePreview(), m.openSelected())
	case act == keymap.QuickLook:
		return m.quickLook()
	}
	return nil
}

func (m *Model) pageSize() int {
	_, _, _, _, _, bodyH := m.previewGeom()
	return max(1, bodyH-1)
}

func (m *Model) scrollPreview(delta int) {
	if m.pv.res == nil {
		return
	}
	_, _, _, _, _, bodyH := m.previewGeom()
	maxScroll := max(0, len(m.pv.res.Lines)-bodyH)
	m.pv.scroll = max(0, min(maxScroll, m.pv.scroll+delta))
}

func (m *Model) quickLook() tea.Cmd {
	e, ok := m.selected()
	if !ok || e.IsDir {
		return nil
	}
	if err := m.deps.Runner.Start(opener.QuickLook(e.Path).Argv); err != nil {
		m.setError(err)
		return nil
	}
	m.setStatus(i18n.T("Abierto en Vista rápida", "Opened in Quick Look"))
	return nil
}

func (m *Model) previewBox() (x, y int, lines []string) {
	x, y, _, _, iw, bodyH := m.previewGeom()
	title := i18n.T("Vista previa", "Preview")
	if e, ok := m.selected(); ok {
		title = e.Name
	}
	body := m.previewBody(iw, bodyH)
	for len(body) < bodyH {
		body = append(body, "")
	}
	facts, hints := "", i18n.T("Espacio/Esc cerrar · ↑↓ otro archivo · ⏎ abrir · Q Vista rápida", "Space/Esc close · ↑↓ other file · ⏎ open · Q Quick Look")
	if m.pv.res != nil {
		facts = strings.Join(m.pv.res.Facts, "  ·  ")
	}
	content := append(body[:bodyH:bodyH], m.styles.dim(fit(facts, iw)), m.styles.dim(fit(hints, iw)))
	return x, y, m.box(title, content, iw)
}

func (m *Model) previewBody(iw, bodyH int) []string {
	r := m.pv.res
	if r == nil {
		return []string{m.styles.dim(i18n.T("Cargando…", "Loading…"))}
	}
	switch r.Kind {
	case preview.KindText, preview.KindDir:
		return m.textBody(r, iw, bodyH)
	case preview.KindImage:
		return m.imageBody(r, iw)
	case preview.KindBinary, preview.KindArchive, preview.KindMedia:
		return m.textBody(r, iw, bodyH)
	case preview.KindOther:
		return []string{m.styles.dim(i18n.T("Archivo especial: sin vista previa", "Special file: not previewed"))}
	case preview.KindError:
		return []string{m.styles.errText(i18n.T("Error: ", "Error: ") + r.Err.Error())}
	}
	return nil
}

func (m *Model) textBody(r *preview.Result, iw, bodyH int) []string {
	end := min(len(r.Lines), m.pv.scroll+bodyH)
	var out []string
	for _, l := range r.Lines[m.pv.scroll:end] {
		out = append(out, strings.ReplaceAll(l, "\t", "    "))
	}
	if r.Truncated && end == len(r.Lines) && len(out) > 0 {
		out[len(out)-1] = m.styles.dim(i18n.T("… recortado", "… truncated"))
	}
	return out
}

func (m *Model) imageBody(r *preview.Result, iw int) []string {
	switch {
	case m.pv.img != nil:
		lines := termimg.Placeholders(m.pv.img.id, m.pv.img.cols, m.pv.img.rows)
		pad := strings.Repeat(" ", max(0, (iw-m.pv.img.cols)/2))
		for i := range lines {
			lines[i] = pad + lines[i]
		}
		return lines
	case m.imagesAllowed:
		return []string{m.styles.dim(i18n.T("Cargando imagen…", "Loading image…"))}
	}
	why := i18n.T("Esta terminal no muestra imágenes. Pulsa Q para Vista rápida.", "This terminal does not show images. Press Q for Quick Look.")
	if m.deps.InTmux {
		why = i18n.T("tmux no deja pasar las imágenes. Pulsa Q para Vista rápida.", "tmux does not pass images through. Press Q for Quick Look.")
	}
	return []string{
		m.styles.dim(fmt.Sprintf(i18n.T("Imagen %d x %d", "Image %d x %d"), r.ImageW, r.ImageH)),
		m.styles.dim(why),
	}
}

// ── Open With menu

func (m *Model) openMenu() tea.Cmd {
	e, ok := m.selected()
	if !ok || e.IsDir {
		return nil
	}
	m.menu = menuState{open: true, path: e.Path, items: m.resolve.Choices(e.Path)}
	return nil
}

func (m *Model) menuGeom() (x, y, iw int) {
	iw = 28
	for _, c := range m.menu.items {
		iw = max(iw, ansi.StringWidth(c.Name)+ansi.StringWidth(c.Argv[0])+6)
	}
	iw = min(iw, m.w-8)
	h := len(m.menu.items) + 2
	return (m.w - iw - 4) / 2, (m.h - h) / 2, iw
}

func (m *Model) menuBox() (x, y int, lines []string) {
	x, y, iw := m.menuGeom()
	var content []string
	for i, c := range m.menu.items {
		pointer := "  "
		if i == m.menu.sel {
			pointer = "▸ "
		}
		line := fit(pointer+c.Name, iw-12) + m.styles.dim(fit(c.Argv[0], 12))
		if i == m.menu.sel {
			line = m.styles.selected(ansi.Strip(line))
		}
		content = append(content, line)
	}
	return x, y, m.box(i18n.T("Abrir con", "Open With"), content, iw)
}

func (m *Model) menuKey(key string, act keymap.Action) tea.Cmd {
	n := len(m.menu.items)
	switch {
	case act == keymap.Quit:
		return m.quit()
	case key == "esc" || act == keymap.OpenWith:
		m.menu = menuState{}
	case act == keymap.Up:
		m.menu.sel = wrap(m.menu.sel-1, n)
	case act == keymap.Down:
		m.menu.sel = wrap(m.menu.sel+1, n)
	case act == keymap.First:
		m.menu.sel = 0
	case act == keymap.Last:
		m.menu.sel = max(0, n-1)
	case key == "enter":
		return m.chooseOpener(m.menu.sel)
	}
	return nil
}

func (m *Model) chooseOpener(i int) tea.Cmd {
	c := m.menu.items[i]
	m.menu = menuState{}
	return m.run(c)
}

func (m *Model) menuClick(x, y int) tea.Cmd {
	mx, my, bx := m.menuBox()
	if !inBox(x, y, mx, my, bx) {
		return nil // like lazymark: a popup closes only with Esc
	}
	if i := y - my - 1; i >= 0 && i < len(m.menu.items) {
		m.menu.sel = i
		return m.chooseOpener(i)
	}
	return nil
}
