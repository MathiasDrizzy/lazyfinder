package ui

import (
	"strings"
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"

	tea "charm.land/bubbletea/v2"
)

// The action icons of the tree (new file, new folder, refresh, collapse all) and the button that switches between
// the normal window and the sidebar mode, like herdr's sidebar and lazymark's Kanban button.
const (
	actionNewFile = iota
	actionNewFolder
	actionCollapse
	actionCount
)

// chipGap is the empty cell between two buttons, so their boxes never touch (order 07, revision 2, S1).
const chipGap = 1

// chipW is the cells of one button of the top rows: a space, the glyph and a space. A Nerd Font glyph is drawn two
// cells wide, so it gets one more cell and the box keeps one cell of margin on each side of it, as the tab chips do.
func (m *Model) chipW() int {
	if m.icons.Style == icons.Nerd {
		return 4
	}
	return 3
}

// chip is the text of a button of chipW cells around glyph g.
func (m *Model) chip(g string) string {
	if m.icons.Style == icons.Nerd {
		return " " + g + "  "
	}
	return " " + g + " "
}

// The buttons at the right of the top row, right to left: the gear at the edge, the mode button and, in the window,
// the Search field, each one chipGap from the next.
func (m *Model) gearX() int  { return m.w - m.gearW() }
func (m *Model) modeX() int  { return m.gearX() - chipGap - m.chipW() }
func (m *Model) fieldX() int { return m.modeX() - chipGap - m.fieldW() }

// actionsWidth is the cells n action buttons take with the gaps between them.
func (m *Model) actionsWidth(n int) int {
	if n <= 0 {
		return 0
	}
	return n*m.chipW() + (n-1)*chipGap
}

type actionSpan struct{ x, y, w, idx int }

// actionDef is one icon of the header: the glyph in both icon styles, its tooltip and what a click does.
type actionDef struct {
	nerd, plain string
	label       func() string
	run         func() tea.Cmd
}

// actionDefs are the icons of the header: the file actions on the Files tab and in the window, the git ones on the Git tab.
func (m *Model) actionDefs() []actionDef {
	if m.projectMode() && m.projTab == tabGit {
		return []actionDef{
			{"⎇", "⎇", func() string { return i18n.T("Ramas: B", "Branches: B") }, m.openGitMenu},
			{"↻", "↻", func() string { return i18n.T("Traer cambios: f", "Fetch: f") }, m.gitFetch},
			{"↓", "↓", func() string { return i18n.T("Bajar (pull): p", "Pull: p") }, m.gitPull},
			{"↑", "↑", func() string { return i18n.T("Subir (push): P", "Push: P") }, m.gitPush},
		}
	}
	return []actionDef{
		{"\uea7f", "+", func() string { return i18n.T("Archivo nuevo: n", "New file: n") }, m.startNewFile},
		{"\uea80", "▢", func() string { return i18n.T("Carpeta nueva: N", "New folder: N") }, m.startNewFolder},
		{"\ueac5", "⊟", func() string { return i18n.T("Contraer todo", "Collapse all") }, m.collapseAll},
	}
}

func (m *Model) collapseAll() tea.Cmd {
	clear(m.expanded)
	m.sel, m.top = 0, 0
	return m.reload()
}

func (m *Model) actionGlyph(idx int) string {
	d := m.actionDefs()[idx]
	if m.icons.Style == icons.Nerd {
		return d.nerd
	}
	return d.plain
}

func (m *Model) modeGlyph() string {
	if m.icons.Style == icons.Nerd {
		return ""
	}
	return "▥"
}

// modeLabel is the tooltip of the mode button.
func (m *Model) modeLabel() string {
	if m.projectMode() {
		return i18n.T("Salir del modo lateral: ctrl+b", "Leave the sidebar mode: ctrl+b")
	}
	return i18n.T("Modo lateral: ctrl+b", "Sidebar mode: ctrl+b")
}

func (m *Model) actionLabel(idx int) string { return m.actionDefs()[idx].label() }

func (m *Model) modeButton() string { return m.styles.dim(m.chip(m.modeGlyph())) }

// actionsLinger is how long the tree buttons of the sidebar mode stay after the pointer last moved, like the title bar
// buttons of the herdr sidebar: a terminal says nothing when the pointer leaves, so activity over the pane shows them
// and they fade a few seconds after the last movement.
const actionsLinger = 3 * time.Second

// actionsFadeMsg wakes the screen up when the buttons are due to fade.
type actionsFadeMsg struct{}

// actionsVisible says whether the buttons of the header are drawn: always in the toolbar of the window, and in the
// Files tab of the sidebar mode only while the pointer is active.
func (m *Model) actionsVisible() bool {
	if m.projectMode() && m.projTab == tabGit {
		return m.git.ok // outside a repository there is nothing for the git buttons to do
	}
	if !m.projectMode() {
		return true
	}
	return m.mouse && m.hover.on && m.deps.Now().Sub(m.hover.at) < actionsLinger
}

// wake asks for an actionsFadeMsg after d.
func (m *Model) wake(d time.Duration) tea.Cmd {
	if m.deps.Wake != nil {
		return m.deps.Wake(d)
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return actionsFadeMsg{} })
}

// fadeActions keeps the wake-up going while the pointer moved again after the one that was scheduled.
func (m *Model) fadeActions() tea.Cmd {
	if left := actionsLinger - m.deps.Now().Sub(m.hover.at); m.hover.on && left > 0 {
		return m.wake(left)
	}
	return nil
}

// actionsShown is how many action icons fit.
func (m *Model) actionsShown() int {
	if !m.actionsVisible() {
		return 0
	}
	n := len(m.actionDefs())
	if m.projectMode() {
		for n > 0 && m.actionsWidth(n) > m.listW()-10 {
			n--
		}
		return n
	}
	if m.w >= 90 {
		return n
	}
	return 0
}

// actionSpans says where each action icon is: on the header of the tree in the sidebar mode, in the toolbar otherwise.
func (m *Model) actionSpans() []actionSpan {
	n := m.actionsShown()
	if n == 0 {
		return nil
	}
	y, end := 0, m.fieldX()-chipGap
	if m.projectMode() {
		y, end = m.headerRow(), m.w
	}
	x := end - m.actionsWidth(n)
	spans := make([]actionSpan, 0, n)
	for i := 0; i < n; i++ {
		spans = append(spans, actionSpan{x: x + i*(m.chipW()+chipGap), y: y, w: m.chipW(), idx: i})
	}
	return spans
}

// actionsText draws the action icons as one string of actionsWidth cells, dimmed in both modes like the title bar
// buttons of the herdr sidebar: the pointer lights the one under it.
func (m *Model) actionsText() string {
	parts := make([]string, m.actionsShown())
	for i := range parts {
		parts[i] = m.styles.dim(m.chip(m.actionGlyph(i)))
	}
	return strings.Join(parts, strings.Repeat(" ", chipGap))
}

// runAction does what an action icon says.
func (m *Model) runAction(idx int) tea.Cmd {
	if defs := m.actionDefs(); idx >= 0 && idx < len(defs) {
		return defs[idx].run()
	}
	return nil
}

// tooltip is the text of the bottom row while the pointer rests on an icon with no label of its own.
func (m *Model) tooltip() string {
	if !m.hover.on || !m.mouse || m.prompt.kind != promptNone {
		return "" // a prompt keeps the bottom row: it says what is being typed
	}
	switch z := m.hit(m.hover.x, m.hover.y); z.kind {
	case zoneAction:
		return " " + m.actionLabel(z.idx)
	case zoneProjectTab:
		return " " + projectTabLabel(projectTab(z.idx))
	case zoneMode:
		return " " + m.modeLabel()
	case zoneSettings:
		return " " + i18n.T("Ajustes: ,", "Settings: ,")
	case zoneBranch:
		return " " + i18n.T("Cambiar de rama: B", "Switch branch: B")
	case zoneScmBtn:
		return " " + scmLabel(z.act)
	}
	return ""
}

// projectTabLabel is the tooltip of a tab: the tabs show only their icon, like the herdr sidebar.
func projectTabLabel(t projectTab) string {
	switch t {
	case tabSearch:
		return i18n.T("Buscar: /", "Search: /")
	case tabGit:
		return "Git"
	}
	return i18n.T("Archivos", "Files")
}

func (m *Model) withTooltip(lines []string) []string {
	if t := m.tooltip(); t != "" && len(lines) > 0 {
		lines[len(lines)-1] = m.styles.accent(fit(t, m.w))
	}
	return lines
}
