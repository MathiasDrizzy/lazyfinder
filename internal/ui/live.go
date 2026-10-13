package ui

import (
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
)

// dirChangedMsg says the content of dir changed on disk and the listing should be read again.
type dirChangedMsg struct{ dir string }

// gitState is the repository status of the current folder; ok is false outside a repository or with git turned off.
type gitState struct {
	st git.Status
	ok bool
}

// waitEvent blocks until the watcher reports a change, then delivers it as a message.
func waitEvent(w watch.Watcher) tea.Cmd {
	ch := w.Events()
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return dirChangedMsg{dir: ev.Dir}
	}
}

// follow moves the watch to the current folder.
func (m *Model) follow() {
	if m.watched == m.dir() {
		return
	}
	if m.watched != "" {
		m.deps.Watcher.Unwatch(m.watched)
	}
	m.watched = ""
	if err := m.deps.Watcher.Watch(m.dir()); err == nil {
		m.watched = m.dir()
	}
}

// onDirChanged re-reads the folder that changed, keeping the selection by name and marking what is new.
func (m *Model) onDirChanged(msg dirChangedMsg) tea.Cmd {
	next := waitEvent(m.deps.Watcher)
	if msg.dir != m.dir() {
		return next
	}
	if _, err := os.Stat(m.dir()); err != nil {
		return tea.Batch(next, m.leaveVanished())
	}
	before := map[string]bool{}
	for _, e := range m.entries() {
		before[e.Name] = true
	}
	cmd := m.reload()
	for _, e := range m.entries() {
		if !before[e.Name] {
			m.fresh[e.Name] = true
		}
	}
	return tea.Batch(next, cmd)
}

// leaveVanished goes to the closest parent that still exists.
func (m *Model) leaveVanished() tea.Cmd {
	d := filepath.Dir(m.dir())
	for {
		if _, err := os.Stat(d); err == nil || filepath.Dir(d) == d {
			return m.enter(d, "")
		}
		d = filepath.Dir(d)
	}
}

func (m *Model) isFresh(name string) bool { return m.fresh[name] }

// readGit refreshes the repository status for dir and drops ignored entries when asked to.
func (m *Model) readGit(l fsmodel.Listing) fsmodel.Listing {
	m.git = gitState{}
	if !m.deps.Cfg.Git.Enabled {
		return l
	}
	st, ok, err := git.Read(l.Dir)
	if err != nil || !ok {
		return l
	}
	m.git = gitState{st: st, ok: true}
	if !m.hideIgnored {
		return l
	}
	kept := make([]fsmodel.Entry, 0, len(l.Entries))
	for _, e := range l.Entries {
		if st.Of(e.Path, e.IsDir) != git.Ignored {
			kept = append(kept, e)
		}
	}
	l.Entries = kept
	return l
}

func (m *Model) gitCode(e fsmodel.Entry) git.Code {
	if !m.git.ok {
		return git.None
	}
	return m.git.st.Of(e.Path, e.IsDir)
}
