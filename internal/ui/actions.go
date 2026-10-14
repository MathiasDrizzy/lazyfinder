package ui

import (
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"

	tea "charm.land/bubbletea/v2"
)

// The action icons of the tree (new file, new folder, refresh, collapse all) and the button that switches between
// the normal window and the sidebar mode, like herdr's sidebar and lazymark's Kanban button.
const (
	actionNewFile = iota
	actionNewFolder
	actionRefresh
	actionCollapse
	actionCount
)

// modeBtnW is the cells of the mode button, left of the gear.
const modeBtnW = 3

// actionW is the cells of one action icon: a space and the glyph.
const actionW = 2

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
		{"\U000f0224", "+", func() string { return i18n.T("Archivo nuevo: n", "New file: n") }, m.startNewFile},
		{"\U000f0257", "▢", func() string { return i18n.T("Carpeta nueva: N", "New folder: N") }, m.startNewFolder},
		{"\U000f0450", "↻", func() string { return i18n.T("Actualizar: r", "Refresh: r") }, m.reload},
		{"\U000f0ab5", "⊟", func() string { return i18n.T("Contraer todo", "Collapse all") }, m.collapseAll},
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

func (m *Model) modeButton() string { return m.styles.accent(" " + m.modeGlyph() + " ") }

// actionsShown is how many action icons fit.
func (m *Model) actionsShown() int {
	n := len(m.actionDefs())
	if m.projectMode() {
		return max(0, min(n, (m.listW()-2-8)/actionW))
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
	y, end := 0, m.w-m.gearW()-modeBtnW-m.fieldW()-1
	if m.projectMode() {
		y, end = 2, m.w-1
	}
	spans := make([]actionSpan, 0, n)
	for i := 0; i < n; i++ {
		spans = append(spans, actionSpan{x: end - (n-i)*actionW, y: y, w: actionW, idx: i})
	}
	return spans
}

// actionsText draws the action icons as one string of n*actionW cells.
func (m *Model) actionsText() string {
	var b strings.Builder
	for i := 0; i < m.actionsShown(); i++ {
		b.WriteString(m.styles.accent(" " + m.actionGlyph(i)))
	}
	return b.String()
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
	if !m.hover.on || !m.mouse {
		return ""
	}
	switch z := m.hit(m.hover.x, m.hover.y); z.kind {
	case zoneAction:
		return " " + m.actionLabel(z.idx)
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

func (m *Model) withTooltip(lines []string) []string {
	if t := m.tooltip(); t != "" && len(lines) > 0 {
		lines[len(lines)-1] = m.styles.accent(fit(t, m.w))
	}
	return lines
}
