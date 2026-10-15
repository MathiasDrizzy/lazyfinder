package ui

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

// herdrPoll is how often the pane of the herdr plugin asks herdr for the folder of the pane next to it.
const herdrPoll = 500 * time.Millisecond

// herdrDirMsg carries the folder of the herdr pane next to the sidebar.
type herdrDirMsg struct{ dir string }

// pollHerdr asks herdr for that folder after d; nil outside herdr.
func (m *Model) pollHerdr(d time.Duration) tea.Cmd {
	ask := m.deps.HerdrDir
	if ask == nil {
		return nil
	}
	if d == 0 {
		return func() tea.Msg { return herdrDirMsg{dir: ask()} }
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return herdrDirMsg{dir: ask()} })
}

// followHerdr goes to the folder of the pane next to the sidebar when it changed since the last answer, so a folder
// picked by hand stays until that pane changes folder.
func (m *Model) followHerdr(dir string) tea.Cmd {
	next := m.pollHerdr(herdrPoll)
	if dir == "" || dir == m.herdrSeen {
		return next
	}
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return next
	}
	m.herdrSeen = dir
	if dir == m.dir() {
		return next
	}
	return tea.Batch(m.enter(dir, ""), next)
}
