package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
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

// projectTabsLine draws the tabs and says where each is, so the click lands on the one drawn there. Labels go away
// (all but the active tab's) when the window is too narrow for them.
func (m *Model) projectTabsLine() (string, []tabSpan) {
	type tab struct{ glyph string }
	nerd := m.icons.Style == icons.Nerd
	tabs := []tab{{"▤"}, {lens}, {"⎇"}}
	if nerd {
		tabs[0].glyph, tabs[1].glyph, tabs[2].glyph = "\uf07b", "\uf002", "\uf126" // the glyphs of the herdr sidebar
	}
	active := tabFiles
	switch {
	case m.searchOn():
		active = tabSearch
	case m.projTab == tabGit:
		active = tabGit
	}
	// Only the icon, like the herdr sidebar: two spaces on each side make a square button that is easy to hit, and
	// the hover covers the same cells as the click.
	var b strings.Builder
	var spans []tabSpan
	x := 0
	for i, t := range tabs {
		text := "  " + t.glyph + "  "
		style := func(s string) string { return s } // readable, not dimmed: the icon is all the tab has
		if projectTab(i) == active {
			style = m.styles.selected
		}
		spans = append(spans, tabSpan{x: x, w: ansi.StringWidth(text)})
		b.WriteString(style(text))
		x += ansi.StringWidth(text)
	}
	return b.String(), spans
}

// projectToolbar is row 0 of the project mode: the tabs and, at the right edge, the Settings gear.
func (m *Model) projectToolbar() string {
	line, _ := m.projectTabsLine()
	pad := m.w - ansi.StringWidth(line) - m.gearW() - modeBtnW
	if pad < 0 {
		return fit(line, m.w-m.gearW()-modeBtnW) + m.modeButton() + m.gearButton() + " "
	}
	return line + strings.Repeat(" ", pad) + m.modeButton() + m.gearButton() + " "
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
	if n == 0 {
		return m.styles.accent(fit(" "+m.projectName(), m.listW()))
	}
	return m.styles.accent(fit(" "+m.projectName(), m.listW()-n*actionW-1)) + m.actionsText() + " "
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
	l := " " + branch + " " + m.git.st.Branch + fmt.Sprintf("  ↓%d ↑%d", m.git.st.Behind, m.git.st.Ahead)
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
