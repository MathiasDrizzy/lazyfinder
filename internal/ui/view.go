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
	"github.com/charmbracelet/x/ansi"
)

const (
	sidebarWide   = 22
	sidebarNarrow = 18
	colGap        = 2
	prefixW       = 4 // pointer, space, folder marker, space
	minNameW      = 16
	crumbSep      = " › "
)

var columnLabels = map[fsmodel.Column]string{
	fsmodel.ColName: "Name", fsmodel.ColModified: "Date Modified", fsmodel.ColCreated: "Date Created",
	fsmodel.ColSize: "Size", fsmodel.ColKind: "Kind",
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

func (m *Model) listX() int { return m.sidebarW() }
func (m *Model) listW() int { return m.w - m.sidebarW() }

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
		w := m.listW() - prefixW
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
	specs := []colSpec{{col: fsmodel.ColName, x: prefixW, w: max(1, nameW())}}
	x := prefixW + specs[0].w + colGap
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
	return e.Name
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
	section("Favorites", m.deps.Cfg.Sidebar.Favorites)
	out = append(out, place{})
	section("Locations", m.deps.Cfg.Sidebar.Locations)
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
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	v.WindowTitle = "lazyfinder"
	return v
}

func (m *Model) tooSmall() string {
	lines := make([]string, max(m.h, 1))
	w := max(m.w, 1)
	lines[0] = ansi.Truncate("Terminal too small", w, "")
	if len(lines) > 1 {
		lines[1] = ansi.Truncate(fmt.Sprintf("Need %dx%d, have %dx%d", minWidth, minHeight, m.w, m.h), w, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) screen() []string {
	lines := make([]string, m.h)
	lines[0] = m.toolbar()
	side := m.sidebarLines()
	list := m.listLines()
	for y := 1; y <= m.h-3; y++ {
		lines[y] = side[y-1] + list[y-1]
	}
	lines[m.h-2] = m.breadcrumb()
	lines[m.h-1] = m.statusLine()
	if m.menu.open {
		x, y, bx := m.menuBox()
		lines = overlay(lines, bx, x, y)
	}
	if m.pv.open {
		x, y, bx := m.previewBox()
		lines = overlay(lines, bx, x, y)
	}
	return lines
}

func (m *Model) toolbar() string {
	back, fwd := " ‹ ", " › "
	if !m.hist.CanBack() {
		back = m.styles.dim(back)
	}
	if !m.hist.CanForward() {
		fwd = m.styles.dim(fwd)
	}
	title := m.styles.bold(filepath.Base(m.dir()))
	if m.dir() == "/" {
		title = m.styles.bold("/")
	}
	count := fmt.Sprintf("%d items", len(m.entries()))
	left := back + fwd + "  " + title
	pad := m.w - ansi.StringWidth(left) - ansi.StringWidth(count) - 1
	if pad < 1 {
		return fit(left, m.w)
	}
	return left + strings.Repeat(" ", pad) + m.styles.dim(count) + " "
}

func (m *Model) sidebarLines() []string {
	w := m.sidebarW()
	rows := m.h - 3
	out := make([]string, rows)
	ps := m.places()
	dests := destinations(ps)
	for i := range out {
		out[i] = strings.Repeat(" ", w-1) + m.styles.border("│")
		if i == 0 {
			continue // aligned with the column header
		}
		pi := i - 1
		if pi >= len(ps) {
			continue
		}
		p := ps[pi]
		var cell string
		switch {
		case p.title != "":
			cell = m.styles.dim(fit(" "+strings.ToUpper(p.title[:1])+p.title[1:], w-1))
		case p.name == "":
			continue
		default:
			cursor := " "
			if m.focus == focusSidebar && len(dests) > 0 && dests[min(m.sideSel, len(dests)-1)] == pi {
				cursor = "▸"
			}
			label := fit(cursor+" "+p.name, w-1)
			switch {
			case p.missing:
				label = m.styles.dim(label)
			case p.path == m.dir():
				label = m.styles.bold(label)
			}
			cell = label
		}
		out[i] = cell + m.styles.border("│")
	}
	return out
}

func (m *Model) listLines() []string {
	rows := m.h - 3
	out := make([]string, rows)
	specs := m.columns()
	hdr := strings.Repeat(" ", prefixW)
	for i, s := range specs {
		label := columnLabels[s.col]
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
	out[0] = m.styles.bold(fit(hdr, m.listW()))
	for r := 0; r < rows-1; r++ {
		idx := m.top + r
		if idx >= len(m.entries()) {
			out[r+1] = strings.Repeat(" ", m.listW())
			continue
		}
		out[r+1] = m.row(idx, specs)
	}
	if len(m.entries()) == 0 && rows > 2 {
		out[2] = fit("   This folder is empty", m.listW())
	}
	return out
}

func (m *Model) row(idx int, specs []colSpec) string {
	e := m.entries()[idx]
	selected := idx == m.sel
	pointer, marker := " ", " "
	if selected {
		pointer = "▸"
	}
	if e.IsDir {
		marker = ">"
	}
	var b strings.Builder
	b.WriteString(pointer + " " + marker + " ")
	for i, s := range specs {
		if i > 0 {
			b.WriteString(strings.Repeat(" ", colGap))
		}
		text := fit(m.cell(e, s.col), s.w)
		if s.col == fsmodel.ColName && e.IsDir && !selected {
			text = m.styles.folder(text)
		}
		b.WriteString(text)
	}
	line := fit(b.String(), m.listW())
	if selected && m.focus == focusList {
		return m.styles.selected(ansi.Strip(line))
	}
	return line
}

func (m *Model) breadcrumb() string {
	var b strings.Builder
	b.WriteString(" ")
	cs := m.crumbs()
	if x := cs[0].x; x > 1 {
		b.WriteString("…" + crumbSep)
	}
	for i, c := range cs {
		if i > 0 {
			b.WriteString(crumbSep)
		}
		if i == len(cs)-1 {
			b.WriteString(m.styles.bold(c.label))
		} else {
			b.WriteString(c.label)
		}
	}
	return fit(b.String(), m.w)
}

const hint = "↑↓ move  ⏎ open  Space preview  O open with  s sort  . hidden  Tab sidebar  q quit"

func (m *Model) statusLine() string {
	switch {
	case m.status == "":
		return m.styles.dim(fit(" "+hint, m.w))
	case m.statusErr:
		return m.styles.errText(fit(" Error: "+m.status, m.w))
	}
	return fit(" "+m.status, m.w)
}
