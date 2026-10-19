package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// pickItem is one line of the picker: what it says, a dim note on the right, and what choosing it does.
type pickItem struct {
	label, hint string
	match       string // more words a filter matches besides the label, such as an action's name in the config
	run         func() tea.Cmd
	del         func() // removes the item for good; nil when it cannot be removed
}

// pickerState is a small list popup the features of milestone 6 share (recent folders, saved searches,
// commands). Enter or a click runs the item; Esc or a click outside closes it.
type pickerState struct {
	open  bool
	title string
	items []pickItem
	sel   int

	rebuild  func() []pickItem                // items again after one was removed
	shortcut func(key string) (tea.Cmd, bool) // a key that picks an item at once (the letter of a mark)

	// filter makes what is typed narrow the list (the command palette): query is the text, all every item.
	filter bool
	query  string
	all    []pickItem
}

func (m *Model) openPicker(title string, items []pickItem) {
	if len(items) == 0 {
		return
	}
	m.picker = pickerState{open: true, title: title, items: items}
}

func (m *Model) pickerGeom() (x, y, iw int) {
	iw = 36
	for _, it := range m.picker.items {
		iw = max(iw, ansi.StringWidth(it.label)+ansi.StringWidth(it.hint)+6)
	}
	iw = min(iw, m.w-8)
	h := m.pickerHead() + m.pickerRows() + 2
	return (m.w - iw - 4) / 2, (m.h - h) / 2, iw
}

// pickerRows is how many rows of items the popup has; the list scrolls to keep the selection visible. A filtered list
// keeps one row when nothing matches, to say so.
func (m *Model) pickerRows() int { return max(1, min(len(m.picker.items), m.h-10-m.pickerHead())) }

// pickerHead is the row the query takes above the items of a filtered list.
func (m *Model) pickerHead() int {
	if m.picker.filter {
		return 1
	}
	return 0
}

// pickerTop is the first item on screen.
func (m *Model) pickerTop() int {
	return max(0, min(m.picker.sel-m.pickerRows()+1, len(m.picker.items)-m.pickerRows()))
}

func (m *Model) pickerBox() (x, y int, lines []string) {
	x, y, iw := m.pickerGeom()
	rows := m.pickerRows()
	top := m.pickerTop()
	var content []string
	if m.picker.filter {
		content = append(content, fit("› "+m.picker.query+"▏", iw))
		if len(m.picker.items) == 0 {
			content = append(content, m.styles.dim(fit("  "+i18n.T("Nada coincide", "Nothing matches"), iw)))
		}
	}
	for i := top; i < min(top+rows, len(m.picker.items)); i++ {
		it := m.picker.items[i]
		pointer := "  "
		if i == m.picker.sel {
			pointer = "▸ "
		}
		hintW := min(ansi.StringWidth(it.hint), iw/2)
		line := fit(pointer+it.label, iw-hintW-1) + " " + m.styles.dim(fit(it.hint, hintW))
		if i == m.picker.sel {
			line = m.styles.selected(ansi.Strip(fit(pointer+it.label, iw-hintW-1) + " " + fit(it.hint, hintW)))
		}
		content = append(content, line)
	}
	return x, y, m.box(m.picker.title, content, iw)
}

func (m *Model) pickerKey(key string, act keymap.Action) tea.Cmd {
	n := len(m.picker.items)
	if m.picker.shortcut != nil {
		if cmd, ok := m.picker.shortcut(key); ok {
			return cmd
		}
	}
	switch {
	case act == keymap.Quit:
		return m.quit()
	case key == "esc":
		m.picker = pickerState{}
	case act == keymap.Up:
		m.picker.sel = wrap(m.picker.sel-1, n)
	case act == keymap.Down:
		m.picker.sel = wrap(m.picker.sel+1, n)
	case act == keymap.First:
		m.picker.sel = 0
	case act == keymap.Last:
		m.picker.sel = n - 1
	case key == "delete" || key == "ctrl+d":
		m.removePick()
	case key == "enter":
		return m.choosePick(m.picker.sel)
	}
	return nil
}

func (m *Model) removePick() {
	it := m.picker.items[m.picker.sel]
	if it.del == nil || m.picker.rebuild == nil {
		return
	}
	it.del()
	rebuild := m.picker.rebuild
	items := rebuild()
	if len(items) == 0 {
		m.picker = pickerState{}
		return
	}
	m.picker.items, m.picker.sel = items, min(m.picker.sel, len(items)-1)
}

func (m *Model) choosePick(i int) tea.Cmd {
	it := m.picker.items[i]
	m.picker = pickerState{}
	return it.run()
}

func (m *Model) pickerClick(x, y int) tea.Cmd {
	px, py, bx := m.pickerBox()
	if !inBox(x, y, px, py, bx) {
		return nil // like lazymark: a popup closes only with Esc
	}
	row := y - py - 1 - m.pickerHead()
	if i := m.pickerTop() + row; row >= 0 && i < len(m.picker.items) {
		m.picker.sel = i
		return m.choosePick(i)
	}
	return nil
}
