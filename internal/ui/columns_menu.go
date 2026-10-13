package ui

import (
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// columnMenu is the menu of a right click on the header: every column with a check when it is shown, and, for the
// column under the pointer, moving it left or right.
func (m *Model) columnMenu(under fsmodel.Column) []ctxItem {
	shown := m.deps.Cfg.Columns
	var items []ctxItem
	current := config.ColumnPreset(shown)
	for _, p := range []struct{ key, label string }{{"basic", i18n.T("Básico", "Basic")}, {"full", i18n.T("Completo", "Full")}} {
		mark := "  "
		if current == p.key {
			mark = "• "
		}
		items = append(items, ctxItem{mark + p.label, func() tea.Cmd { m.applyColumnPreset(p.key); return nil }})
	}
	for _, c := range fsmodel.AllColumns {
		if c == fsmodel.ColName {
			continue
		}
		label, mark := columnLabel(c), "  "
		if slices.Contains(shown, string(c)) {
			mark = "✓ "
		}
		items = append(items, ctxItem{mark + label, func() tea.Cmd { m.toggleColumn(c); return nil }})
	}
	if under != fsmodel.ColName && slices.Contains(shown, string(under)) {
		items = append(items,
			ctxItem{i18n.T("Mover a la izquierda", "Move left"), func() tea.Cmd { m.moveColumn(under, -1); return nil }},
			ctxItem{i18n.T("Mover a la derecha", "Move right"), func() tea.Cmd { m.moveColumn(under, 1); return nil }})
	}
	return items
}

// applyColumnPreset shows the columns of a preset ("basic" or "full") and saves them.
func (m *Model) applyColumnPreset(name string) {
	m.setColumns(slices.Clone(config.ColumnPresets[name]))
	if !slices.Contains(m.deps.Cfg.Columns, string(m.sort.By)) {
		s := m.sort
		s.By, s.Descending = fsmodel.ColName, false
		m.resort(s)
	}
}

func (m *Model) setColumns(cols []string) {
	m.deps.Cfg.Columns = cols
	m.saveSize("columns", cols)
	m.ensureVisible()
}

// toggleColumn shows a hidden column at the end or hides a shown one. Name cannot be hidden.
func (m *Model) toggleColumn(c fsmodel.Column) {
	cols := slices.Clone(m.deps.Cfg.Columns)
	if i := slices.Index(cols, string(c)); i >= 0 {
		cols = slices.Delete(cols, i, i+1)
	} else {
		cols = append(cols, string(c))
	}
	if !slices.Contains(cols, "name") {
		cols = append([]string{"name"}, cols...)
	}
	m.setColumns(cols)
	if m.sort.By == c && !slices.Contains(cols, string(c)) {
		s := m.sort
		s.By, s.Descending = fsmodel.ColName, false
		m.resort(s) // the list cannot stay sorted by a column nobody sees
	}
}

// moveColumn swaps a column with its neighbour; the name always stays first.
func (m *Model) moveColumn(c fsmodel.Column, delta int) {
	cols := slices.Clone(m.deps.Cfg.Columns)
	i := slices.Index(cols, string(c))
	j := i + delta
	if i < 0 || j < 0 || j >= len(cols) || cols[j] == string(fsmodel.ColName) {
		return
	}
	cols[i], cols[j] = cols[j], cols[i]
	m.setColumns(cols)
}
