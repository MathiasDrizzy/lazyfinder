package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

const (
	sidebarWide   = 22
	sidebarNarrow = 18
	colGap        = 2
	prefixW       = 4 // pointer, space, folder marker, space
	gitGutterW    = 2 // git state and a space, only inside a repository
	minNameW      = 16
	crumbSep      = " › "
)

func columnLabel(c fsmodel.Column) string {
	switch c {
	case fsmodel.ColModified:
		return i18n.T("Fecha de modificación", "Date Modified")
	case fsmodel.ColCreated:
		return i18n.T("Fecha de creación", "Date Created")
	case fsmodel.ColSize:
		return i18n.T("Tamaño", "Size")
	case fsmodel.ColKind:
		return i18n.T("Clase", "Kind")
	}
	return i18n.T("Nombre", "Name")
}

var columnWidths = map[fsmodel.Column]int{
	fsmodel.ColModified: 18, fsmodel.ColCreated: 18, fsmodel.ColSize: 10, fsmodel.ColKind: 18,
}

type colSpec struct {
	col  fsmodel.Column
	x, w int // x is relative to the list's left edge
}

func (m *Model) sidebarW() int {
	if m.w >= 90 {
		return sidebarWide
	}
	return sidebarNarrow
}

// The screen, top to bottom: toolbar (row 0), the two panels (rows 1..h-3), the breadcrumb (h-2) and the key bar (h-1).
// Each panel has a border, so the content of the list starts one cell in and its header sits on row 2.
func (m *Model) listX() int { return m.sidebarW() + 1 } // first content column of the list panel
func (m *Model) listW() int { return m.w - m.sidebarW() - 2 }

const (
	headerRow = 2 // column header
	firstRow  = 3 // first entry
)

// prefix is the width before the name: pointer, folder marker and, inside a repository, the git state.
func (m *Model) prefix() int {
	if m.git.ok {
		return prefixW + gitGutterW
	}
	return prefixW
}

func (m *Model) panelH() int { return m.h - 3 }

// dropOrder says which optional columns go first when the window is narrow.
var dropOrder = []fsmodel.Column{fsmodel.ColCreated, fsmodel.ColKind, fsmodel.ColSize, fsmodel.ColModified}

// columns lays out the visible columns; optional ones are dropped by dropOrder until the name fits.
func (m *Model) columns() []colSpec {
	var extra []fsmodel.Column
	for _, c := range m.deps.Cfg.Columns {
		if col := fsmodel.Column(c); col != fsmodel.ColName {
			extra = append(extra, col)
		}
	}
	nameW := func() int {
		w := m.listW() - m.prefix()
		for _, c := range extra {
			w -= columnWidths[c] + colGap
		}
		return w
	}
	for _, drop := range dropOrder {
		if nameW() >= minNameW {
			break
		}
		extra = slices.DeleteFunc(extra, func(c fsmodel.Column) bool { return c == drop })
	}
	specs := []colSpec{{col: fsmodel.ColName, x: m.prefix(), w: max(1, nameW())}}
	x := m.prefix() + specs[0].w + colGap
	for _, c := range extra {
		specs = append(specs, colSpec{col: c, x: x, w: columnWidths[c]})
		x += columnWidths[c] + colGap
	}
	return specs
}

func (m *Model) cell(e fsmodel.Entry, c fsmodel.Column) string {
	switch c {
	case fsmodel.ColModified:
		return fsmodel.FormatDate(e.Modified, m.deps.Now())
	case fsmodel.ColCreated:
		return fsmodel.FormatDate(e.Created, m.deps.Now())
	case fsmodel.ColSize:
		if e.IsDir {
			return "--"
		}
		return fsmodel.HumanSize(e.Size)
	case fsmodel.ColKind:
		return e.Kind
	}
	if m.searchOn() && m.search.recursive {
		if w := m.where(e); w != "" {
			return strings.Repeat("  ", e.Depth) + e.Name + "  ‹ " + w
		}
	}
	return strings.Repeat("  ", e.Depth) + e.Name
}

// place is a flattened sidebar line: a section title or a destination.
type place struct {
	title   string
	name    string
	path    string
	missing bool
}

func (m *Model) places() []place {
	var out []place
	section := func(title string, ps []config.Place) {
		out = append(out, place{title: title})
		for _, p := range ps {
			_, err := os.Stat(p.Path)
			out = append(out, place{name: p.Name, path: p.Path, missing: err != nil})
		}
	}
	section(i18n.T("Favoritos", "Favorites"), m.deps.Cfg.Sidebar.Favorites)
	out = append(out, place{})
	section(i18n.T("Ubicaciones", "Locations"), m.deps.Cfg.Sidebar.Locations)
	return out
}

// destinations returns the indexes into places() that can be selected.
func destinations(ps []place) []int {
	var idx []int
	for i, p := range ps {
		if p.title == "" && p.name != "" {
			idx = append(idx, i)
		}
	}
	return idx
}

type crumb struct {
	label, path string
	x, w        int
}

// crumbs lays out the breadcrumb for the bottom bar, dropping leading segments (shown as "…") when it is too long.
func (m *Model) crumbs() []crumb {
	dir := m.dir()
	var parts []crumb
	cur := dir
	for {
		base := filepath.Base(cur)
		if parent := filepath.Dir(cur); parent == cur {
			parts = append([]crumb{{label: "/", path: cur}}, parts...)
			break
		}
		parts = append([]crumb{{label: base, path: cur}}, parts...)
		cur = filepath.Dir(cur)
	}
	width := func(ps []crumb, ellipsis bool) int {
		w := 1
		if ellipsis {
			w += 1 + ansi.StringWidth(crumbSep)
		}
		for i, p := range ps {
			if i > 0 {
				w += ansi.StringWidth(crumbSep)
			}
			w += ansi.StringWidth(p.label)
		}
		return w
	}
	ellipsis := false
	for len(parts) > 1 && width(parts, ellipsis) > m.w-1 {
		parts, ellipsis = parts[1:], true
	}
	x := 1
	if ellipsis {
		x += 1 + ansi.StringWidth(crumbSep)
	}
	for i := range parts {
		if i > 0 {
			x += ansi.StringWidth(crumbSep)
		}
		parts[i].x, parts[i].w = x, ansi.StringWidth(parts[i].label)
		x += parts[i].w
	}
	return parts
}

// View renders the screen.
func (m *Model) View() tea.View {
	var content string
	if m.w < minWidth || m.h < minHeight {
		content = m.tooSmall()
	} else {
		content = strings.Join(m.screen(), "\n")
		if m.styles.bg != nil {
			content = theme.PaintBackground(content, m.styles.bg)
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	if m.mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.WindowTitle = "lazyfinder"
	return v
}

func (m *Model) tooSmall() string {
	lines := make([]string, max(m.h, 1))
	w := max(m.w, 1)
	lines[0] = ansi.Truncate(i18n.T("Terminal demasiado pequeña", "Terminal too small"), w, "")
	if len(lines) > 1 {
		lines[1] = ansi.Truncate(fmt.Sprintf(i18n.T("Hace falta %dx%d, hay %dx%d", "Need %dx%d, have %dx%d"), minWidth, minHeight, m.w, m.h), w, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) screen() []string {
	lines := make([]string, m.h)
	lines[0] = m.toolbar()
	side := m.panel(i18n.T("Sitios", "Places"), "", m.sidebarLines(), m.sidebarW(), m.panelH(), m.focus == focusSidebar)
	list := m.panel(m.folderTitle(), m.counter(), m.listLines(), m.w-m.sidebarW(), m.panelH(), m.focus == focusList)
	for y := 0; y < m.panelH(); y++ {
		lines[y+1] = side[y] + list[y]
	}
	lines[m.h-2] = m.breadcrumb()
	lines[m.h-1] = m.footerBar()
	if m.menu.open {
		x, y, bx := m.menuBox()
		lines = overlay(lines, bx, x, y)
	}
	if m.pv.open {
		x, y, bx := m.previewBox()
		lines = overlay(lines, bx, x, y)
	}
	if m.set.open {
		x, y, bx := m.settingsBox()
		lines = overlay(lines, bx, x, y)
	}
	return lines
}

func (m *Model) folderTitle() string {
	if m.dir() == "/" {
		return "/"
	}
	return filepath.Base(m.dir())
}

// counter is the "n of m" in the bottom border of the list panel.
func (m *Model) counter() string {
	n := len(m.entries())
	if n == 0 {
		return fmt.Sprintf(i18n.T("%d de %d", "%d of %d"), 0, 0)
	}
	return fmt.Sprintf(i18n.T("%d de %d", "%d of %d"), m.sel+1, n)
}

const (
	gear = "⚙"
	lens = "⌕"
)

func (m *Model) toolbar() string {
	back, fwd := " ‹ ", " › "
	if !m.hist.CanBack() {
		back = m.styles.dim(back)
	}
	if !m.hist.CanForward() {
		fwd = m.styles.dim(fwd)
	}
	left := back + fwd + "  " + m.styles.bold(m.folderTitle())
	right := m.styles.dim(fmt.Sprintf(i18n.T("%d elementos", "%d items"), len(m.entries()))) + "  " + m.styles.accent(lens) + "  " + m.styles.accent(gear) + " "
	if b := m.branchLabel(); b != "" {
		right = m.styles.accent(b) + "  " + right
	}
	pad := m.w - ansi.StringWidth(left) - ansi.StringWidth(right)
	if pad < 1 {
		return fit(left, m.w-4) + " " + m.styles.accent(gear) + " "
	}
	return left + strings.Repeat(" ", pad) + right
}

func (m *Model) sidebarLines() []string {
	w := m.sidebarW() - 2
	rows := m.panelH() - 2
	out := make([]string, rows)
	ps := m.places()
	dests := destinations(ps)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
		if i >= len(ps) {
			continue
		}
		p := ps[i]
		switch {
		case p.title != "":
			out[i] = m.styles.dim(fit(" "+p.title, w))
		case p.name == "":
		default:
			label := fit("  "+p.name, w)
			focused := m.focus == focusSidebar && len(dests) > 0 && dests[min(m.sideSel, len(dests)-1)] == i
			switch {
			case focused:
				label = m.styles.selected(ansi.Strip(fit("▸ "+p.name, w)))
			case p.missing:
				label = m.styles.dim(label)
			case p.path == m.dir():
				label = m.styles.bold(label)
			}
			out[i] = label
		}
	}
	return out
}

func (m *Model) listLines() []string {
	rows := m.panelH() - 2
	out := make([]string, rows)
	specs := m.columns()
	hdr := strings.Repeat(" ", m.prefix())
	for i, s := range specs {
		label := columnLabel(s.col)
		if s.col == m.sort.By {
			if m.sort.Descending {
				label += " ▼"
			} else {
				label += " ▲"
			}
		}
		if i > 0 {
			hdr += strings.Repeat(" ", colGap)
		}
		hdr += fit(label, s.w)
	}
	out[0] = m.styles.title(fit(hdr, m.listW()))
	for r := 1; r < rows; r++ {
		idx := m.top + r - 1
		if idx >= len(m.entries()) {
			out[r] = strings.Repeat(" ", m.listW())
			continue
		}
		out[r] = m.row(idx, specs)
	}
	if len(m.entries()) == 0 && rows > 2 {
		out[2] = m.styles.dim(fit("   "+i18n.T("Esta carpeta está vacía", "This folder is empty"), m.listW()))
	}
	return out
}

func (m *Model) row(idx int, specs []colSpec) string {
	e := m.entries()[idx]
	selected := idx == m.sel
	pointer, marker := " ", " "
	switch {
	case selected:
		pointer = "▸"
	case m.isFresh(e.Path):
		pointer = "+"
	}
	if e.IsDir {
		marker = ">"
		if m.expanded[e.Path] {
			marker = "v"
		}
	}
	var b strings.Builder
	b.WriteString(pointer + " " + marker + " ")
	if m.git.ok {
		code := string(m.gitCode(e))
		if code == "" {
			code = " "
		}
		b.WriteString(code + " ")
	}
	for i, s := range specs {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", colGap))
		}
		text := fit(m.cell(e, s.col), s.w)
		switch {
		case s.col != fsmodel.ColName || selected:
		case m.isFresh(e.Path):
			text = m.styles.fresh(text)
		case e.IsDir:
			text = m.styles.folder(text)
		}
		b.WriteString(text)
	}
	line := fit(b.String(), m.listW())
	switch {
	case selected && m.focus == focusList:
		return m.styles.selected(ansi.Strip(line))
	case selected:
		return m.styles.selectedInactive(ansi.Strip(line))
	}
	return line
}

func (m *Model) breadcrumb() string {
	var b strings.Builder
	b.WriteString(" ")
	cs := m.crumbs()
	if x := cs[0].x; x > 1 {
		b.WriteString(m.styles.dim("…" + crumbSep))
	}
	for i, c := range cs {
		if i > 0 {
			b.WriteString(m.styles.dim(crumbSep))
		}
		if i == len(cs)-1 {
			b.WriteString(m.styles.bold(c.label))
		} else {
			b.WriteString(m.styles.title(c.label))
		}
	}
	return fit(b.String(), m.w)
}

// footerItems are the entries of the key bar, in priority order of survival: the last ones are dropped first when the
// window is narrow, except Quit, which always stays.
var footerItems = []struct {
	action keymap.Action
	label  func() string
}{
	{keymap.Open, func() string { return i18n.T("Abrir", "Open") }},
	{keymap.Preview, func() string { return i18n.T("Vista previa", "Preview") }},
	{keymap.OpenWith, func() string { return i18n.T("Abrir con", "Open with") }},
	{keymap.Search, func() string { return i18n.T("Buscar", "Search") }},
	{keymap.Sort, func() string { return i18n.T("Ordenar", "Sort") }},
	{keymap.Hidden, func() string { return i18n.T("Ocultos", "Hidden") }},
	{keymap.Focus, func() string { return i18n.T("Barra lateral", "Sidebar") }},
	{keymap.Settings, func() string { return i18n.T("Ajustes", "Settings") }},
	{keymap.Quit, func() string { return i18n.T("Salir", "Quit") }},
}

// keyLabel is how a key is written in the key bar.
func keyLabel(k string) string {
	switch k {
	case "enter":
		return "⏎"
	case "space":
		return "Space"
	case "tab":
		return "Tab"
	case "pgup":
		return "PgUp"
	case "pgdown":
		return "PgDn"
	}
	return k
}

func (m *Model) footerBar() string {
	if m.goto_.open {
		if m.status != "" && m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		return m.gotoBar()
	}
	if m.searchOn() {
		if m.status != "" && m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		bar, _ := m.searchBar()
		return bar
	}
	if m.status != "" {
		if m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		return fit(" "+m.status, m.w)
	}
	build := func(skip int) (string, int) {
		var parts []string
		for i, it := range footerItems {
			if i >= len(footerItems)-skip-1 && it.action != keymap.Quit && i < len(footerItems)-1 {
				continue
			}
			ks := m.keys.Keys(it.action)
			if len(ks) == 0 {
				continue
			}
			parts = append(parts, m.styles.footer(it.label()+": ")+m.styles.footerKey(keyLabel(ks[0])))
		}
		line := " " + strings.Join(parts, m.styles.dim(" | "))
		return line, ansi.StringWidth(line)
	}
	for skip := 0; skip < len(footerItems); skip++ {
		if line, w := build(skip); w <= m.w {
			return fit(line, m.w)
		}
	}
	line, _ := build(len(footerItems))
	return fit(line, m.w)
}

// branchLabel is the git branch with its distance from the upstream, empty outside a repository.
func (m *Model) branchLabel() string {
	if !m.git.ok || m.git.st.Branch == "" {
		return ""
	}
	l := "⎇ " + m.git.st.Branch
	if m.git.st.Ahead > 0 {
		l += fmt.Sprintf(" ↑%d", m.git.st.Ahead)
	}
	if m.git.st.Behind > 0 {
		l += fmt.Sprintf(" ↓%d", m.git.st.Behind)
	}
	return l
}
