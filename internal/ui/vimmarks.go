package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const marksFile = "marks.json"

// isMarkName says whether key names a mark: one letter or digit, as in Vim.
func isMarkName(key string) bool {
	if len(key) != 1 {
		return false
	}
	c := key[0]
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

// startMark waits for the letter that marks the current folder.
func (m *Model) startMark() tea.Cmd {
	m.awaitMark = true
	m.setStatus(i18n.T("Marcar la carpeta con una letra (Esc cancela)", "Mark the folder with a letter (Esc cancels)"))
	return nil
}

// markKey takes the key after m: a letter or a digit marks the current folder with it, anything else cancels.
func (m *Model) markKey(key string) tea.Cmd {
	m.awaitMark = false
	if !isMarkName(key) {
		return nil
	}
	if m.vimMarks == nil {
		m.vimMarks = map[string]string{}
	}
	m.vimMarks[key] = m.dir()
	saveState(m.stateFile(marksFile), m.vimMarks)
	m.setStatus(fmt.Sprintf(i18n.T("Marca %s: %s", "Mark %s: %s"), key, m.homeShort(m.dir())))
	return nil
}

// showMarks lists the marks; Enter or the mark's own letter goes to its folder, delete removes it.
func (m *Model) showMarks() tea.Cmd {
	items := m.markItems()
	if len(items) == 0 {
		m.setStatus(i18n.T("Aún no hay marcas: m y una letra marcan la carpeta", "No marks yet: m and a letter mark the folder"))
		return nil
	}
	m.openPicker(i18n.T("Marcas", "Marks"), items)
	m.picker.rebuild = m.markItems
	m.picker.shortcut = func(key string) (tea.Cmd, bool) {
		path, ok := m.vimMarks[key]
		if !ok {
			return nil, false
		}
		m.picker = pickerState{}
		return m.goMark(path), true
	}
	return nil
}

func (m *Model) markItems() []pickItem {
	names := make([]string, 0, len(m.vimMarks))
	for k := range m.vimMarks {
		names = append(names, k)
	}
	slices.Sort(names)
	items := make([]pickItem, 0, len(names))
	for _, k := range names {
		path := m.vimMarks[k]
		items = append(items, pickItem{
			label: k + "  " + filepath.Base(path), hint: m.homeShort(filepath.Dir(path)),
			run: func() tea.Cmd { return m.goMark(path) },
			del: func() {
				delete(m.vimMarks, k)
				saveState(m.stateFile(marksFile), m.vimMarks)
			},
		})
	}
	return items
}

func (m *Model) goMark(path string) tea.Cmd {
	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		m.setError(fmt.Errorf(i18n.T("La carpeta de la marca ya no está: %s", "The folder of the mark is gone: %s"), m.homeShort(path)))
		return nil
	}
	return m.enter(path, "")
}
