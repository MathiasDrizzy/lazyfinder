package ui

import (
	"time"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/charmbracelet/x/ansi"
)

const maxAutoWidth = 34 // an automatic column never takes more than this

// date writes a time in the date format of the config.
func (m *Model) date(t time.Time) string {
	return fsmodel.FormatDateAs(m.deps.Cfg.DateFormat, t, m.deps.Now())
}

// dateWidth is the widest a date can get in the current format and language.
func (m *Model) dateWidth() int {
	now := m.deps.Now()
	w := 0
	for _, t := range []time.Time{now, now.AddDate(0, 0, -1), now.AddDate(-1, 0, 0)} {
		w = max(w, ansi.StringWidth(m.date(t)))
	}
	return w
}

// colWidth is the width of an optional column: the one in config.json, else what its title and its content need.
func (m *Model) colWidth(c fsmodel.Column) int {
	if w, ok := m.deps.Cfg.ColumnWidths[string(c)]; ok {
		return w
	}
	return m.autoWidth(c)
}

// contentWidth is what the cells of a column need, without its title: the narrowest it gets before its text is cut.
// A width set in config.json is kept.
func (m *Model) contentWidth(c fsmodel.Column) int {
	if w, ok := m.deps.Cfg.ColumnWidths[string(c)]; ok {
		return w
	}
	need := 0
	switch c {
	case fsmodel.ColModified, fsmodel.ColCreated, fsmodel.ColOpened:
		need = m.dateWidth()
	case fsmodel.ColPerms:
		need = 10
	case fsmodel.ColSize:
		need = columnWidths[c]
	default:
		for _, e := range m.entries() {
			need = max(need, ansi.StringWidth(m.cell(e, c))+1)
		}
	}
	return max(config.MinColumn, min(need, m.colWidth(c)))
}

// shrinkFloor is the narrowest a column gets before it is dropped: dates, sizes and permissions are never cut, and a
// text column such as Kind keeps textFloor cells.
func (m *Model) shrinkFloor(c fsmodel.Column) int {
	switch c {
	case fsmodel.ColModified, fsmodel.ColCreated, fsmodel.ColOpened, fsmodel.ColPerms, fsmodel.ColSize:
		return m.contentWidth(c)
	}
	return min(m.contentWidth(c), max(config.MinColumn, textFloor))
}

const textFloor = 12 // a text column cut to this still says what it holds ("Hoja de cál…")

// autoWidth fits a column to its title (with room for the sort arrow) and its widest cell, so "Kind" and "Date
// Modified" are not cut while the window allows it.
func (m *Model) autoWidth(c fsmodel.Column) int {
	need := ansi.StringWidth(columnLabel(c)) + 2
	switch c {
	case fsmodel.ColModified, fsmodel.ColCreated, fsmodel.ColOpened:
		need = max(need, m.dateWidth())
	case fsmodel.ColPerms:
		need = max(need, 10)
	case fsmodel.ColSize:
		need = max(need, columnWidths[c])
	default:
		for _, e := range m.entries() {
			need = max(need, ansi.StringWidth(m.cell(e, c))+1) // one cell of air before the border
		}
	}
	return max(config.MinColumn, min(need, maxAutoWidth))
}
