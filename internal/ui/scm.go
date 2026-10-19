package ui

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// The Git tab of the sidebar mode is a quick look at what changed, like the one in herdr's sidebar: the changes that
// are staged and the ones that are not, the branch on top, and the diff of a file. It changes nothing (Mathias,
// 2026-10-14: "no quiero hacer commit en un sidebar … solo ver que es lo que cambio"): staging, discarding,
// stashing, switching branches, pulling and pushing are left to lazygit (L opens it).

type scmKind int

const (
	scmStagedHdr scmKind = iota
	scmChangesHdr
	scmStashHdr
	scmFile
	scmStash
	scmNone // the line that says there is nothing to show
)

type scmRow struct {
	kind   scmKind
	ch     git.Change
	staged bool
	n      int // for a header: how many files are under it
}

// scmAct is what a button of a row does.
type scmAct int

const scmActOpen scmAct = 0 // the only one: show the diff

type scmBtn struct {
	x, w int
	act  scmAct
}

type scmState struct {
	sel, top int
	folded   [3]bool // the staged section, the changes section, the stashes (folded until opened)
}

const scmBtnW = 3 // the glyph with a space on each side: every button has the same width and the glyph in the middle

func (m *Model) scmOn() bool { return m.projectMode() && m.projTab == tabGit && !m.searchOn() }

func (m *Model) scmRows() []scmRow {
	if !m.git.ok {
		return nil
	}
	var rows []scmRow
	section := func(kind scmKind, list []git.Change, staged bool) {
		if len(list) == 0 {
			return
		}
		rows = append(rows, scmRow{kind: kind, n: len(list)})
		if m.scm.folded[kind] {
			return
		}
		for _, c := range list {
			rows = append(rows, scmRow{kind: scmFile, ch: c, staged: staged})
		}
	}
	if len(m.git.st.Staged) == 0 && len(m.git.st.Unstaged) == 0 {
		rows = append(rows, scmRow{kind: scmNone})
	}
	section(scmStagedHdr, m.git.st.Staged, true)
	section(scmChangesHdr, m.git.st.Unstaged, false)
	if len(m.git.stashes) > 0 {
		rows = append(rows, scmRow{kind: scmStashHdr, n: len(m.git.stashes)})
		if !m.scm.folded[scmStashHdr] {
			for i, s := range m.git.stashes {
				_, msg, _ := strings.Cut(s, ": ")
				rows = append(rows, scmRow{kind: scmStash, n: i, ch: git.Change{Path: msg}})
			}
		}
	}
	return rows
}

// scmVisible is how many rows fit under the branch line.
func (m *Model) scmVisible() int { return max(1, m.panelH()-3) }

func (m *Model) scmClamp() []scmRow {
	rows := m.scmRows()
	m.scm.sel = max(0, min(m.scm.sel, len(rows)-1))
	vis := m.scmVisible()
	m.scm.top = max(0, min(m.scm.top, max(0, len(rows)-vis)))
	if m.scm.sel < m.scm.top {
		m.scm.top = m.scm.sel
	}
	if m.scm.sel >= m.scm.top+vis {
		m.scm.top = m.scm.sel - vis + 1
	}
	return rows
}

func (k scmKind) header() bool { return k <= scmStashHdr }

func scmGlyph(scmAct) string { return "↗" }

func scmLabel(scmAct) string { return i18n.T("↗ Abrir los cambios", "↗ Open changes") }

func scmActs(r scmRow) []scmAct {
	if r.kind != scmFile {
		return nil
	}
	return []scmAct{scmActOpen}
}

// scmRowY is the screen line of row i, or -1 when it is scrolled out.
func (m *Model) scmRowY(i int) int {
	if i < m.scm.top || i >= m.scm.top+m.scmVisible() {
		return -1
	}
	return m.firstRow() + i - m.scm.top
}

// scmButtonsVisible says whether row i wears its buttons: the selected row and the one under the pointer do.
func (m *Model) scmButtonsVisible(i int) bool {
	if i == m.scm.sel {
		return true
	}
	return m.mouse && m.hover.on && m.hover.y == m.scmRowY(i) && m.hover.x >= m.listX() && m.hover.x < m.listX()+m.listW()
}

// scmButtons says where the buttons of row i are, right-aligned; none while the row does not show them.
func (m *Model) scmButtons(i int) []scmBtn {
	rows := m.scmRows()
	if i < 0 || i >= len(rows) || !m.scmButtonsVisible(i) {
		return nil
	}
	acts := scmActs(rows[i])
	end := m.listX() + m.listW()
	out := make([]scmBtn, 0, len(acts))
	for j, a := range acts {
		out = append(out, scmBtn{x: end - (len(acts)-j)*scmBtnW, w: scmBtnW, act: a})
	}
	return out
}

func (m *Model) scmHeaderBranch() string {
	if !m.git.ok || m.git.st.Branch == "" {
		return " " + i18n.T("sin git", "no git")
	}
	l := " ⎇ " + m.git.st.BranchLabel()
	if m.git.st.Changed() > 0 {
		l += "*"
	}
	if m.git.st.Behind > 0 || m.git.st.Ahead > 0 {
		l += fmt.Sprintf(" ↓%d ↑%d", m.git.st.Behind, m.git.st.Ahead)
	}
	return l
}

func (m *Model) scmLines() []string {
	rows := m.scmClamp()
	h := m.panelH() - 2
	out := make([]string, h)
	w := m.listW()
	n := m.actionsShown()
	out[0] = m.styles.accent(fit(m.scmHeaderBranch(), w-m.actionsWidth(n)-edgeW)) + m.actionsText() + strings.Repeat(" ", edgeW)
	for r := 1; r < h; r++ {
		i := m.scm.top + r - 1
		if i >= len(rows) {
			out[r] = strings.Repeat(" ", w)
			continue
		}
		out[r] = m.scmRowLine(i, rows[i])
	}
	switch {
	case !m.git.ok && h > 2:
		out[2] = m.styles.dim(fit("  "+i18n.T("No es un repositorio git", "Not a git repository"), w))
	}
	return out
}

func (m *Model) scmRowLine(i int, r scmRow) string {
	w := m.listW()
	btns := m.scmButtons(i)
	room := w
	if len(btns) > 0 {
		room = btns[0].x - m.listX()
	}
	var line string
	switch {
	case r.kind == scmNone:
		line = m.styles.dim(fit("  "+i18n.T("Sin cambios", "No changes"), room))
	case r.kind == scmStash:
		line = fit(" ⇩ "+shownName(r.ch.Path), room)
		if i != m.scm.sel {
			line = m.styles.dim(line)
		}
	case r.kind != scmFile:
		arrow, title := "▾", i18n.T("Cambios preparados", "Staged Changes")
		if m.scm.folded[r.kind] {
			arrow = "▸"
		}
		switch r.kind {
		case scmChangesHdr:
			title = i18n.T("Cambios", "Changes")
		case scmStashHdr:
			title = "Stashes"
		}
		count := fmt.Sprintf(" %d", r.n)
		line = fit(" "+arrow+" "+title, room-len(count)-1) + count
		line = fit(line, room)
		if i != m.scm.sel {
			line = m.styles.accent(line)
		}
	default:
		code := r.ch.Code
		name := shownName(path.Base(r.ch.Path))
		dir := shownName(path.Dir(r.ch.Path))
		lead := " "
		if m.iconW() > 0 {
			lead += m.iconCell(m.icons.Pick(icons.Item{Name: name})) + " "
		}
		letterW := 3
		if len(btns) > 0 {
			letterW = 0
		}
		text := fit(lead+name, room-letterW)
		if dir != "." {
			free := room - letterW - ansi.StringWidth(lead+name) - 1
			if free > 3 {
				text = lead + name + " " + fit(dir, free)
				text = fit(text, room-letterW)
			}
		}
		if i != m.scm.sel {
			if dir != "." && ansi.StringWidth(lead+name)+2 < room-letterW {
				rest := ansi.Cut(text, ansi.StringWidth(lead+name), room)
				line = m.styles.gitPaint(string(code), ansi.Truncate(text, ansi.StringWidth(lead+name), "")) + m.styles.dim(rest)
			} else {
				line = m.styles.gitPaint(string(code), text)
			}
		} else {
			line = text
		}
		if letterW > 0 {
			letter := " " + m.styles.gitLetter(string(code)) + " "
			if i == m.scm.sel {
				letter = " " + string(code) + " "
			}
			line += letter
		}
	}
	if i == m.scm.sel {
		line = m.selectedStyle()(ansi.Strip(line))
	}
	for _, b := range btns {
		g := " " + scmGlyph(b.act) + " "
		if i == m.scm.sel {
			line += m.selectedStyle()(g)
		} else {
			line += m.styles.accent(g)
		}
	}
	return fit(line, w)
}

// scmKey handles the keys of the Git tab; ok is false for keys that mean the same as anywhere else.
func (m *Model) scmKey(key string, act keymap.Action) (tea.Cmd, bool) {
	rows := m.scmClamp()
	n := len(rows)
	page := m.scmVisible()
	switch key {
	case "o":
		return m.scmOpenFile(), true
	case "L":
		return m.openLazygit(), true
	}
	switch act {
	case keymap.Up:
		m.scm.sel = wrap(m.scm.sel-1, n)
	case keymap.Down:
		m.scm.sel = wrap(m.scm.sel+1, n)
	case keymap.PageUp:
		m.scm.sel = max(0, m.scm.sel-page)
	case keymap.PageDown:
		m.scm.sel = max(0, min(n-1, m.scm.sel+page))
	case keymap.First:
		m.scm.sel = 0
	case keymap.Last:
		m.scm.sel = max(0, n-1)
	case keymap.Open, keymap.Preview, keymap.Expand:
		return m.scmActivate(), true
	case keymap.Parent, keymap.Collapse:
		if n > 0 && rows[m.scm.sel].kind.header() {
			m.scm.folded[rows[m.scm.sel].kind] = true
		}
	case keymap.Help, keymap.Quit, keymap.Reload, keymap.Settings, keymap.Project, keymap.Search, keymap.Commands:
		return nil, false
	}
	m.scmClamp()
	return nil, true // anything else would act on the hidden folder tree
}

// scmActivate opens the diff of the selected file, or folds the selected section.
func (m *Model) scmActivate() tea.Cmd {
	rows := m.scmRows()
	if m.scm.sel >= len(rows) {
		return nil
	}
	r := rows[m.scm.sel]
	switch {
	case r.kind.header():
		m.scm.folded[r.kind] = !m.scm.folded[r.kind]
		return nil
	case r.kind != scmFile:
		return nil
	}
	return m.openDiff(r.ch, r.staged)
}

// scmOpenFile opens the selected file with its program.
func (m *Model) scmOpenFile() tea.Cmd {
	rows := m.scmRows()
	if m.scm.sel >= len(rows) || rows[m.scm.sel].kind != scmFile {
		return nil
	}
	r := rows[m.scm.sel]
	if r.ch.Code == git.Deleted {
		m.setStatus(i18n.T("El archivo está borrado", "The file is deleted"))
		return nil
	}
	abs := filepath.Join(m.git.st.Root, filepath.FromSlash(r.ch.Path))
	m.noteVisit(abs)
	return m.run(m.resolve.Resolve(abs))
}

// scmClickRow selects a row; a second click on a file shows its diff and one on a section folds it.
func (m *Model) scmClickRow(i int, double bool) tea.Cmd {
	rows := m.scmRows()
	if i < 0 || i >= len(rows) {
		return nil
	}
	m.scm.sel = i
	if rows[i].kind.header() {
		m.scm.folded[rows[i].kind] = !m.scm.folded[rows[i].kind]
		return nil
	}
	if rows[i].kind != scmFile {
		return nil
	}
	if double {
		return m.scmActivate()
	}
	return nil
}

// scmClickBtn selects the row of a button and shows its diff, the one thing a button does.
func (m *Model) scmClickBtn(i int, _ scmAct) tea.Cmd {
	m.scm.sel = i
	return m.scmActivate()
}
