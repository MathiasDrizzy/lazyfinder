package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
)

// filtered keeps the entries of l whose name passes the quick filter.
func (m *Model) filtered(l fsmodel.Listing) fsmodel.Listing {
	if m.filter == "" {
		return l
	}
	kept := make([]fsmodel.Entry, 0, len(l.Entries))
	for _, e := range l.Entries {
		if fsmodel.MatchName(m.filter, e.Name) {
			kept = append(kept, e)
		}
	}
	l.Entries = kept
	return l
}

func (m *Model) startFilter() tea.Cmd {
	if m.searchOn() {
		return nil // the list holds search results, not the folder
	}
	m.prompt = promptState{kind: promptFilter, input: m.filter}
	return nil
}

// setFilter applies a new filter to the current folder and keeps the selection when its row survives.
func (m *Model) setFilter(f string) tea.Cmd {
	m.filter = f
	return m.reload()
}

// filterKey edits the filter while the prompt is open: every change filters at once, Enter keeps the filter,
// Esc drops it.
func (m *Model) filterKey(k tea.KeyPressMsg) tea.Cmd {
	switch key := k.String(); key {
	case "esc":
		m.prompt = promptState{}
		return m.setFilter("")
	case "enter":
		m.prompt = promptState{}
		return nil
	case "backspace":
		if r := []rune(m.prompt.input); len(r) > 0 {
			m.prompt.input = string(r[:len(r)-1])
		}
	case "ctrl+u":
		m.prompt.input = ""
	default:
		if k.Text == "" || k.Mod&(tea.ModCtrl|tea.ModAlt) != 0 {
			return nil
		}
		m.prompt.input += k.Text
	}
	return m.setFilter(m.prompt.input)
}
