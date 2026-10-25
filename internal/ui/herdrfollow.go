package ui

import (
	"os"
	"time"

	tea "charm.land/bubbletea/v2"
)

// herdrPoll is how often the pane of the herdr plugin asks herdr for the folder of the pane next to it.
const herdrPoll = 500 * time.Millisecond

const (
	// herdrSlowPoll is how often it asks once herdrQuietPolls answers in a row did not change.
	herdrSlowPoll   = time.Second
	herdrQuietPolls = 20
)

// herdrDelay is the time to the next question to herdr: quick while the folder of the other pane changes or the user
// is here, slower when nothing has changed for a while (eight sidebars, one per tab, ask all day).
func (m *Model) herdrDelay() time.Duration {
	if m.herdrQuiet >= herdrQuietPolls {
		return herdrSlowPoll
	}
	return herdrPoll
}

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
	if dir == "" || dir == m.herdrSeen {
		m.herdrQuiet++
		return m.pollHerdr(m.herdrDelay())
	}
	m.herdrQuiet = 0
	next := m.pollHerdr(herdrPoll)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return next
	}
	m.herdrSeen = dir
	if dir == m.dir() {
		return next
	}
	return tea.Batch(m.enter(dir, ""), next)
}

// herdrFitMsg is the answer of herdr to a change of width.
type herdrFitMsg struct{ err error }

// fitHerdr puts the pane of the herdr plugin at the width of the mode it shows (see Deps.HerdrFit); nil outside the
// plugin.
func (m *Model) fitHerdr(force bool) tea.Cmd {
	fit := m.deps.HerdrFit
	if fit == nil {
		return nil
	}
	s, normal := m.deps.Cfg.Herdr, !m.projectMode()
	return func() tea.Msg { return herdrFitMsg{err: fit(s, normal, force)} }
}

// switchMode is the user changing between the normal and the sidebar mode, which also changes the width of the pane
// of the herdr plugin.
func (m *Model) switchMode() tea.Cmd {
	m.toggleProject()
	return m.fitHerdr(true)
}
