package ui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
)

// The project sidebar mode (--sidebar, ctrl+b) is a narrow panel meant to sit next to another terminal, like the
// herdr sidebar: three tabs on top (Files, Search, Git), the project name in capitals, the tree with the git letter at
// the right edge and the branch with its distance from the upstream at the bottom.

// projectTab is the tab of the project sidebar that is showing.
type projectTab int

const (
	tabFiles projectTab = iota
	tabSearch
	tabGit
)

type tabSpan struct{ x, w int }

func (m *Model) projectMode() bool { return m.deps.Follow }

// activeTab is the tab of the sidebar mode that is showing.
func (m *Model) activeTab() projectTab {
	switch {
	case m.searchOn():
		return tabSearch
	case m.projTab == tabGit:
		return tabGit
	}
	return tabFiles
}

// projectTabsLine draws the middle row of the tab bar and says where each tab is, so the click lands on the one drawn
// there. Like the activity bar of the herdr sidebar: a space, then each icon in a chip with a space between chips.
// The Nerd Font glyphs of the tabs are drawn two cells wide, so their chip keeps a second cell for them (" G  ") and
// the icon sits in the middle of its chip.
func (m *Model) projectTabsLine() (string, []tabSpan) {
	glyphs := []string{"▤", lens, "⎇"}
	slack := ""
	if m.icons.Style == icons.Nerd {
		glyphs = []string{"\uf07b", "\uf002", "\uf126"} // the glyphs of the herdr sidebar
		slack = " "
	}
	var b strings.Builder
	var spans []tabSpan
	b.WriteString(" ")
	x := 1
	for i, g := range glyphs {
		if i > 0 {
			b.WriteString(" ")
			x++
		}
		text := " " + g + slack + " "
		style := m.styles.dim
		if projectTab(i) == m.activeTab() {
			style = m.activeChip
		}
		spans = append(spans, tabSpan{x: x, w: ansi.StringWidth(text)})
		b.WriteString(style(text))
		x += ansi.StringWidth(text)
	}
	return b.String(), spans
}

// toolbarRow is the row of the buttons of the toolbar: the middle row of the tab bar in the sidebar mode.
func (m *Model) toolbarRow() int {
	if m.projectMode() {
		return 1
	}
	return 0
}

// sideButtonX is the first column of the mode button of the sidebar mode; the gear follows it at the right edge.
func (m *Model) sideButtonX() int { return m.modeX() }

// projectToolbar is the tab bar of the sidebar mode, activityH rows: the tabs and, at the right edge, the mode button
// and the Settings gear on the middle row. The chip of the active tab, and of the button under the pointer, reaches
// into the rows above and below with half blocks, a tall button like those of the herdr sidebar.
func (m *Model) projectToolbar() []string {
	line, spans := m.projectTabsLine()
	right := m.modeButton() + strings.Repeat(" ", chipGap) + m.gearButton()
	mid := fit(line, m.sideButtonX()) + right
	type capSpan struct {
		x, w int
		c    color.Color
	}
	var caps []capSpan
	p, colored := theme.Lookup(m.deps.Cfg.Theme)
	if colored {
		a := spans[m.activeTab()]
		caps = append(caps, capSpan{a.x, a.w, chipActiveBg(p)})
		if r, ok := m.hoverRect(); ok && r.y == 1 {
			caps = append(caps, capSpan{r.x, r.w, chipHoverBg(p)})
		}
	}
	edge := func(glyph string) string {
		var b strings.Builder
		x := 0
		for _, c := range caps {
			if c.x < x || c.x+c.w > m.w {
				continue
			}
			b.WriteString(strings.Repeat(" ", c.x-x))
			b.WriteString(lipgloss.NewStyle().Foreground(c.c).Render(strings.Repeat(glyph, c.w)))
			x = c.x + c.w
		}
		return b.String() + strings.Repeat(" ", max(0, m.w-x))
	}
	return []string{edge("▄"), mid, edge("▀")}
}

func (m *Model) projectTabAt(x int) (projectTab, bool) {
	_, spans := m.projectTabsLine()
	for i, s := range spans {
		if x >= s.x && x < s.x+s.w {
			return projectTab(i), true
		}
	}
	return 0, false
}

// projectName is the folder being followed in capitals, the heading of the tree.
func (m *Model) projectName() string {
	return strings.ToUpper(m.folderTitle())
}

// projectHeader is the line above the tree.
func (m *Model) projectHeader() string {
	n := m.actionsShown()
	name := m.styles.folder(fit(" "+m.projectName(), m.listW()-m.actionsWidth(n)))
	if n == 0 {
		return name
	}
	return name + m.actionsText()
}

// projectFooter is the bottom line of the project mode: the branch, and how far it is from its upstream.
func (m *Model) projectFooter() string {
	if !m.git.ok || m.git.st.Branch == "" {
		return m.styles.dim(fit(" "+i18n.T("sin git", "no git"), m.w))
	}
	branch := "⎇"
	if m.icons.Style == icons.Nerd {
		branch = "\ue725"
	}
	l := " " + branch + " " + m.git.st.BranchLabel() + fmt.Sprintf("  ↓%d ↑%d", m.git.st.Behind, m.git.st.Ahead)
	if n := m.git.st.Changed(); n > 0 {
		l += fmt.Sprintf("  ● %d", n)
	}
	return m.styles.accent(fit(l, m.w))
}

// clickProjectTab switches to a tab of the project mode.
func (m *Model) clickProjectTab(t projectTab) {
	switch t {
	case tabSearch:
		m.openSearch()
	case tabGit:
		_ = m.endSearch()
		m.projTab = tabGit
		m.scm = scmState{folded: [3]bool{scmStashHdr: true}}
		_ = m.reload() // reads the stashes
	default:
		_ = m.endSearch()
		m.projTab = tabFiles
		_ = m.reload()
	}
}

// projectFooterItems are the key hints of the narrow mode: only what a 30 column window can hold.
var projectFooterItems = []struct {
	action keymap.Action
	label  func() string
}{
	{keymap.Project, func() string { return i18n.T("Salir del modo", "Exit mode") }},
	{keymap.Search, func() string { return i18n.T("Buscar", "Search") }},
	{keymap.Help, func() string { return i18n.T("Ayuda", "Help") }},
	{keymap.Quit, func() string { return i18n.T("Salir", "Quit") }},
}

// treeRow draws row idx of the sidebar mode with the layout of a row of the herdr sidebar and the colors of the normal
// window (Mathias, 2026-10-14: the two modes must not look like two programs): the indent (two cells a level), the
// > or v of a folder, the icon and the name in the folder style (a git state recolors it), and at the right edge the
// git letter, or a dot for a folder with changes inside, with two cells after it. The selection only sets the
// background, so the icon keeps its color.
func (m *Model) treeRow(idx, width int) string {
	e := m.entries()[idx]
	chevron := "  "
	if e.IsDir {
		chevron = "> "
		if m.expanded[e.Path] {
			chevron = "v "
		}
	}
	lead := strings.Repeat("  ", e.Depth) + chevron
	used := 2*e.Depth + 2
	if m.iconW() > 0 {
		ic := m.icons.Pick(icons.Item{Name: e.Name, IsDir: e.IsDir, IsLink: e.IsLink, Open: m.expanded[e.Path]})
		lead += m.iconCell(ic)
		used += m.iconW()
	}
	code := m.gitCode(e)
	marker := ""
	switch {
	case m.isFresh(e.Path): // what just appeared says so for a few seconds, without relying on its color
		marker = m.styles.fresh("+")
	case !m.git.ok || code == git.None || code == git.Ignored:
	case e.IsDir:
		marker = m.styles.gitPaint(string(code), "●")
	default:
		marker = m.styles.bold(m.styles.gitLetter(string(code)))
	}
	tail := 0
	if marker != "" {
		tail = 4 // a space, the marker and two cells clear of the edge
	}
	plain := strings.TrimPrefix(m.cell(e, fsmodel.ColName), strings.Repeat("  ", e.Depth))
	if m.editing(e.Path) {
		plain = m.editedName()
	}
	name := fit(plain, max(1, width-used-tail))
	switch {
	case m.isFresh(e.Path):
		name = m.styles.fresh(name)
	case code == git.Ignored:
		name = m.styles.dim(name)
	case m.git.ok && code != git.None:
		name = m.styles.gitPaint(string(code), name)
	case e.IsDir:
		name = m.styles.folder(name)
	}
	line := lead + name
	if marker != "" {
		line += strings.Repeat(" ", max(1, width-ansi.StringWidth(line)-3)) + marker + "  "
	}
	line = fit(line, width)
	switch {
	case idx == m.sel:
		return m.rowBackground(line, m.focus == focusList)
	case m.marked[e.Path]:
		return m.rowBackground(line, false)
	}
	return line
}

// rowBackground puts the selection background behind line, the strong one (bold, like the selected row) or the soft
// one, and keeps the colors of the text on it. Without a palette it falls back to the selection styles over the plain
// text.
func (m *Model) rowBackground(line string, strong bool) string {
	p, ok := theme.Lookup(m.deps.Cfg.Theme)
	switch {
	case !ok && strong:
		return m.styles.selected(ansi.Strip(line))
	case !ok:
		return m.styles.selectedInactive(ansi.Strip(line))
	}
	seq := theme.BackgroundSeq(p.Surface0)
	if strong {
		seq = theme.BackgroundSeq(p.Surface1) + "\x1b[1m"
	}
	return "\x1b[0m" + seq + dropBackgrounds(line, seq) + "\x1b[0m"
}
