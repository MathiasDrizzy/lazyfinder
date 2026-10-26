package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
)

// dirChangedMsg says the content of dir changed on disk and the listing should be read again.
type dirChangedMsg struct{ dir string }

// gitState is the repository status of the current folder; ok is false outside a repository or with git turned off.
type gitState struct {
	st      git.Status
	ok      bool
	stashes []string // "stash@{0}: message", read only while the Git tab shows them
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

// follow keeps the watcher on the current folder and on every folder expanded in place.
func (m *Model) follow() {
	want := map[string]bool{m.dir(): true}
	if gd := m.gitDir(); gd != "" {
		want[gd] = true // stage, commit and checkout change files in here, not in the folder
	}
	for _, d := range m.treeDirs {
		want[d] = true
	}
	for d := range m.watchSet {
		if !want[d] {
			m.deps.Watcher.Unwatch(d)
			delete(m.watchSet, d)
		}
	}
	for d := range want {
		if m.watchSet[d] {
			continue
		}
		if err := m.deps.Watcher.Watch(d); err == nil {
			m.watchSet[d] = true
		}
	}
}

// onDirChanged re-reads the folder that changed, keeping the selection by name and marking what is new.
func (m *Model) onDirChanged(msg dirChangedMsg) tea.Cmd {
	next := waitEvent(m.deps.Watcher)
	if !m.watchSet[msg.dir] || m.search.mode != searchOff {
		return next
	}
	if msg.dir != "" && msg.dir == m.gitDir() { // git itself changed: the letters and the branch, not the list
		if sig := gitDirSignature(msg.dir); sig == m.gitSig {
			m.gitRescan = false // a read that was waiting has nothing left to read
			return next         // git only read its files: scanning again would read them again
		}
		if wait := m.deps.GitMinGap - m.deps.Now().Sub(m.gitScanAt); wait > 0 {
			if m.gitRescan {
				return next // the read that waits will see this change too
			}
			m.gitRescan = true
			return tea.Batch(next, tea.Tick(wait, func(time.Time) tea.Msg { return dirChangedMsg{dir: msg.dir} }))
		}
		m.gitRescan = false
		return tea.Batch(next, m.reload()) // reading the list again scans git again
	}
	if _, err := os.Stat(msg.dir); err != nil {
		if msg.dir == m.dir() {
			return tea.Batch(next, m.leaveVanished())
		}
		delete(m.expanded, msg.dir) // an expanded folder that vanished
	}
	before := map[string]bool{}
	for _, e := range m.entries() {
		before[e.Path] = true
	}
	m.fromWatcher, m.gitHeld = true, 0
	cmd := m.reload()
	m.fromWatcher = false
	if wait := m.gitHeld; wait > 0 && !m.gitLate {
		m.gitLate = true
		cmd = tea.Batch(cmd, tea.Tick(wait, func(time.Time) tea.Msg { return gitLateMsg{} }))
	}
	opened := false
	for _, e := range m.entries() {
		if before[e.Path] {
			continue
		}
		m.fresh[e.Path] = true
		if m.deps.Follow && e.IsDir && !e.IsLink && !m.expanded[e.Path] {
			m.expanded[e.Path] = true // a folder an agent just made is shown open
			opened = true
		}
	}
	if opened {
		cmd = tea.Batch(cmd, m.reload())
		for _, e := range m.entries() { // what was inside the new folder is new as well
			if !before[e.Path] {
				m.fresh[e.Path] = true
			}
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

func (m *Model) isFresh(path string) bool { return m.fresh[path] }

// readGit reads a repository; a test swaps it to count the reads.
var readGit = git.Read

// gitDirSignature fingerprints the files of the .git folder (names, sizes, modification times): the watcher fires for
// the folder when git only reads its index, and the marks do not change then.
func gitDirSignature(dir string) string {
	es, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, e := range es {
		if fi, err := e.Info(); err == nil && !fi.IsDir() {
			fmt.Fprintf(&b, "%s|%d|%d;", e.Name(), fi.Size(), fi.ModTime().UnixNano())
		}
	}
	return b.String()
}

// scanGit refreshes the repository status for dir; outside a repository, or with git turned off, ok stays false.
func (m *Model) scanGit(dir string) {
	if wait := m.deps.GitMinGap - m.deps.Now().Sub(m.gitScanAt); m.fromWatcher && m.git.ok && dir == m.listing.Dir && wait > 0 {
		m.gitHeld = wait // the folder changed again inside the gap: the marks stay as they are until it ends
		return
	}
	m.git = gitState{}
	if !m.deps.Cfg.Git.Enabled {
		return
	}
	st, ok, err := readGit(dir)
	if err != nil { // the repository is not read at all (an old git, a timeout): say why the marks are gone
		m.setError(err)
		return
	}
	if ok {
		m.git = gitState{st: st, ok: true}
		if m.projectMode() && m.projTab == tabGit {
			m.git.stashes, _ = git.Stashes(st.Root)
		}
		m.gitSig = gitDirSignature(m.gitDir())
		m.gitScanAt = m.deps.Now()
	}
}

// visible drops the entries git ignores when the user asked to hide them, and .git in the sidebar tree, which hides it
// always, as herdr-sidebar does (tree.rs: "`.git` is always hidden; other dotfiles only when `show_hidden` is off").
func (m *Model) visible(es []fsmodel.Entry) []fsmodel.Entry {
	hideIgnored := m.git.ok && m.hideIgnored
	if !hideIgnored && !m.projectMode() {
		return es
	}
	kept := make([]fsmodel.Entry, 0, len(es))
	for _, e := range es {
		switch {
		case m.projectMode() && e.Name == ".git":
		case hideIgnored && m.git.st.Of(e.Path, e.IsDir) == git.Ignored:
		default:
			kept = append(kept, e)
		}
	}
	return kept
}

func (m *Model) gitCode(e fsmodel.Entry) git.Code {
	if !m.git.ok {
		return git.None
	}
	return m.git.st.Of(e.Path, e.IsDir)
}

// withChildren turns a listing into the visible rows: ignored entries hidden when asked, and the folders expanded in
// place followed by their children, one level deeper each.
func (m *Model) withChildren(l fsmodel.Listing) fsmodel.Listing {
	m.treeDirs = nil
	l.Entries = m.rows(m.visible(l.Entries), 0)
	return l
}

func (m *Model) rows(es []fsmodel.Entry, depth int) []fsmodel.Entry {
	out := make([]fsmodel.Entry, 0, len(es))
	for _, e := range es {
		e.Depth = depth
		out = append(out, e)
		if !e.IsDir || e.IsLink || !m.expanded[e.Path] {
			continue
		}
		child, err := fsmodel.ReadDir(e.Path, fsmodel.ReadOptions{ShowHidden: m.hidden, Sort: m.sort})
		if err != nil {
			delete(m.expanded, e.Path)
			continue
		}
		m.treeDirs = append(m.treeDirs, e.Path)
		out = append(out, m.rows(m.visible(child.Entries), depth+1)...)
	}
	return out
}

// toggle opens or closes the folder at path in place.
func (m *Model) toggle(path string) tea.Cmd {
	if m.searchOn() {
		return nil
	}
	if m.expanded[path] {
		delete(m.expanded, path)
	} else {
		m.expanded[path] = true
	}
	m.sel = max(m.sel, 0)
	return m.reload()
}

// expandSelected opens the selected folder in place.
func (m *Model) expandSelected() tea.Cmd {
	e, ok := m.selected()
	if !ok || !e.IsDir || e.IsLink || m.expanded[e.Path] {
		return nil
	}
	return m.toggle(e.Path)
}

// collapseSelected closes the selected folder; on a row inside an open folder it selects that folder and closes it.
func (m *Model) collapseSelected() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		return nil
	}
	if e.IsDir && m.expanded[e.Path] {
		return m.toggle(e.Path)
	}
	if e.Depth == 0 {
		return nil
	}
	parent := filepath.Dir(e.Path)
	m.sel = 0
	m.selectPath(parent)
	return m.toggle(parent)
}

// gitDir is the .git folder of the repository being shown, empty outside one (a worktree's .git file is not watched).
func (m *Model) gitDir() string {
	if !m.git.ok {
		return ""
	}
	gd := filepath.Join(m.git.st.Root, ".git")
	if fi, err := os.Stat(gd); err != nil || !fi.IsDir() {
		return ""
	}
	return gd
}

// toggleProject enters or leaves the project sidebar mode (the same one --follow starts): a tree of the current
// folder with names only, no places sidebar. The columns of the window come back when it is left.
func (m *Model) toggleProject() {
	if m.deps.Follow {
		m.deps.Follow = false
		m.projTab = tabFiles
		m.deps.Cfg.Columns = m.savedColumns
		clear(m.expanded)
		m.relist()
		return
	}
	m.deps.Follow = true
	m.projTab = tabFiles
	m.savedColumns = m.deps.Cfg.Columns
	m.deps.Cfg.Columns = []string{"name"}
	m.relist() // the tree starts collapsed: nothing opens when the mode changes
}

// relist reads the folder again for a new mode and watches what the new listing shows, and only that.
func (m *Model) relist() {
	if err := m.load(m.dir(), ""); err != nil {
		m.setError(err)
	}
	m.follow()
}

// gitLateMsg asks for the read of git that a burst of changes in the folder held back.
type gitLateMsg struct{}

// onGitLate reads git once the gap that held back the reads caused by a burst of changes in the folder is over.
func (m *Model) onGitLate() tea.Cmd {
	m.gitLate = false
	if m.search.mode != searchOff {
		return nil
	}
	return m.reload()
}
