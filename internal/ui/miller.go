package ui

import (
	"bytes"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/charmbracelet/x/ansi"
)

const (
	layoutList    = "list"
	layoutColumns = "columns"

	millerSnippetBytes = 8 << 10
	millerMaxLines     = 200
)

// millerState holds what the two side columns show. It is refreshed after every update (syncMiller), never while
// drawing, so the view stays a pure function of the model.
type millerState struct {
	parentDir string
	parent    []fsmodel.Entry
	rightPath string
	right     []string
	rightDir  bool
}

func (m *Model) columnsOn() bool { return m.layout == layoutColumns }

// toggleLayout switches between the detail list and the column view, and remembers the choice in config.json.
func (m *Model) toggleLayout() tea.Cmd {
	if m.columnsOn() {
		m.layout = layoutList
	} else {
		m.layout = layoutColumns
	}
	m.deps.Cfg.Layout = m.layout
	if m.deps.ConfigPath != "" {
		if err := config.Update(m.deps.ConfigPath, map[string]any{"layout": m.layout}); err != nil {
			m.setError(err)
		}
	}
	m.mil = millerState{}
	m.syncMiller()
	return nil
}

// syncMiller refreshes the side columns when the folder or the selection changed.
func (m *Model) syncMiller() {
	if !m.columnsOn() {
		return
	}
	parent := filepath.Dir(m.dir())
	if m.mil.parentDir != parent || m.mil.parent == nil {
		m.mil.parentDir, m.mil.parent = parent, nil
		if l, err := fsmodel.ReadDir(parent, fsmodel.ReadOptions{ShowHidden: m.hidden, Sort: m.sort}); err == nil {
			m.mil.parent = l.Entries
		}
	}
	e, ok := m.selected()
	if !ok {
		m.mil.rightPath, m.mil.right = "", nil
		return
	}
	if m.mil.rightPath == e.Path && m.mil.right != nil {
		return
	}
	m.mil.rightPath, m.mil.rightDir = e.Path, e.IsDir
	m.mil.right = millerRight(e, m.hidden, m.sort)
}

// millerRight is the text of the third column: the names inside a folder, or the start of a text file.
func millerRight(e fsmodel.Entry, hidden bool, sort fsmodel.Sort) []string {
	if e.IsDir {
		l, err := fsmodel.ReadDir(e.Path, fsmodel.ReadOptions{ShowHidden: hidden, Sort: sort})
		if err != nil {
			return []string{}
		}
		out := []string{}
		for i, c := range l.Entries {
			if i == millerMaxLines {
				break
			}
			name := c.Name
			if c.IsDir {
				name += "/"
			}
			out = append(out, name)
		}
		return out
	}
	f, _, err := fsmodel.OpenRegular(e.Path)
	if err != nil {
		return []string{fsmodel.KindOf(e.Name, false, e.IsLink)}
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, millerSnippetBytes)
	n, _ := f.Read(buf)
	buf = buf[:n]
	if bytes.IndexByte(buf, 0) >= 0 {
		return []string{fsmodel.KindOf(e.Name, false, e.IsLink), fsmodel.HumanSize(e.Size)}
	}
	lines := strings.Split(strings.ReplaceAll(string(buf), "\r\n", "\n"), "\n")
	out := make([]string, 0, min(len(lines), millerMaxLines))
	for _, l := range lines[:min(len(lines), millerMaxLines)] {
		out = append(out, strings.Map(func(r rune) rune {
			switch {
			case r == '\t':
				return ' '
			case r < 0x20 || (r >= 0x7f && r < 0xa0):
				return '?'
			}
			return r
		}, l))
	}
	return out
}

type millerGeom struct{ parentW, centerW, rightW int }

func (m *Model) millerGeom() millerGeom {
	w := m.listW()
	g := millerGeom{parentW: max(8, w/5), rightW: max(12, w*2/5)}
	if p := m.deps.Cfg.PreviewWidth; p > 0 {
		g.rightW = max(12, min(p, w-g.parentW-12))
	}
	g.centerW = max(10, w-g.parentW-g.rightW-2)
	return g
}

func padTo(s string, w int) string {
	s = ansi.Truncate(showControls(s), w, "…")
	return s + strings.Repeat(" ", max(0, w-ansi.StringWidth(s)))
}

func (m *Model) millerLines() []string {
	rows := m.panelH() - 2
	g := m.millerGeom()
	sep := m.styles.dim("│")
	out := make([]string, rows)
	parent := m.millerParentLines(rows, g.parentW)
	center := m.millerCenterLines(rows, g.centerW)
	right := m.millerRightLines(rows, g.rightW)
	for r := range out {
		out[r] = parent[r] + sep + center[r] + sep + right[r]
	}
	return out
}

func (m *Model) millerParentLines(rows, w int) []string {
	out := make([]string, rows)
	out[0] = m.styles.title(padTo(i18n.T("Carpeta superior", "Parent"), w))
	cur := -1
	for i, e := range m.mil.parent {
		if e.Path == m.dir() {
			cur = i
		}
	}
	top := max(0, cur-(rows-1)/2)
	for r := 1; r < rows; r++ {
		out[r] = strings.Repeat(" ", w)
		if i := top + r - 1; i < len(m.mil.parent) {
			e := m.mil.parent[i]
			name := " " + e.Name
			if e.IsDir {
				name += "/"
			}
			if i == cur {
				out[r] = m.styles.selected(ansi.Strip(padTo(name, w)))
			} else {
				out[r] = m.styles.dim(padTo(name, w))
			}
		}
	}
	return out
}

func (m *Model) millerCenterLines(rows, w int) []string {
	out := make([]string, rows)
	out[0] = m.styles.title(padTo(i18n.T("Nombre", "Name"), w))
	specs := []colSpec{{col: fsmodel.ColName, w: max(1, w-m.prefix())}}
	for r := 1; r < rows; r++ {
		idx := m.top + r - 1
		if idx >= len(m.entries()) {
			out[r] = strings.Repeat(" ", w)
			continue
		}
		out[r] = m.rowW(idx, specs, w)
	}
	if len(m.entries()) == 0 && rows > 2 {
		out[2] = m.styles.dim(padTo("   "+i18n.T("Esta carpeta está vacía", "This folder is empty"), w))
	}
	return out
}

func (m *Model) millerRightLines(rows, w int) []string {
	out := make([]string, rows)
	title := i18n.T("Vista previa", "Preview")
	if e, ok := m.selected(); ok {
		title = e.Name
	}
	out[0] = m.styles.title(padTo(title, w))
	for r := 1; r < rows; r++ {
		out[r] = strings.Repeat(" ", w)
		if i := r - 1; i < len(m.mil.right) {
			line := " " + m.mil.right[i]
			if m.mil.rightDir && strings.HasSuffix(line, "/") {
				out[r] = m.styles.folder(padTo(line, w))
			} else {
				out[r] = m.styles.dim(padTo(line, w))
			}
		}
	}
	return out
}

// millerHit resolves a click in the column layout: the parent column goes there, the middle column selects.
func (m *Model) millerHit(x, y int) zone {
	g := m.millerGeom()
	rel := x - m.listX()
	switch {
	case y < firstRow:
		return zone{}
	case rel < g.parentW:
		i := m.parentTop() + y - firstRow
		if i >= 0 && i < len(m.mil.parent) && m.mil.parent[i].IsDir {
			return zone{kind: zoneCrumb, path: m.mil.parent[i].Path}
		}
	case rel > g.parentW && rel <= g.parentW+g.centerW:
		if idx := m.top + y - firstRow; idx < len(m.entries()) {
			return zone{kind: zoneRow, idx: idx}
		}
	}
	return zone{}
}

func (m *Model) parentTop() int {
	cur := -1
	for i, e := range m.mil.parent {
		if e.Path == m.dir() {
			cur = i
		}
	}
	return max(0, cur-(m.panelH()-3)/2)
}
