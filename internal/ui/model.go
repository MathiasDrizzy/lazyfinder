// Package ui is the Bubble Tea front end. It is the only package that imports the TUI framework
// (docs/adr/0002-architecture.md).
package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/history"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
	uv "github.com/charmbracelet/ultraviolet"
)

const (
	minWidth          = 60
	minHeight         = 16
	doubleClickWindow = 400 * time.Millisecond
	wheelStep         = 3
)

// Deps are the collaborators of the model; tests replace them.
type Deps struct {
	Home    string
	Start   string
	Cfg     config.Config
	Runner  opener.Runner
	Now     func() time.Time
	NoColor bool
	Watcher watch.Watcher // nil: nothing is followed
	Warning string        // shown in the status line at start (for example an invalid config)

	// ConfigPath is where the settings popup saves; empty means it only changes the running session.
	ConfigPath string
}

type focus int

const (
	focusList focus = iota
	focusSidebar
)

// Model is the whole UI state. It is mutated only inside Update.
type Model struct {
	deps    Deps
	styles  styles
	resolve opener.Resolver
	keys    keymap.Map
	mouse   bool
	set     settingsState

	w, h int

	hist     history.History
	listing  fsmodel.Listing
	sort     fsmodel.Sort
	hidden   bool
	sel, top int
	lastSel  map[string]string // directory -> name selected when we left it
	focus    focus
	sideSel  int

	status    string
	statusErr bool

	lastClick     click
	pv            previewState
	menu          menuState
	imagesAllowed bool // the terminal answered the Kitty query (or config forces it)
	nextImgID     int

	watched     string          // the folder the watcher follows
	fresh       map[string]bool // names that appeared while we looked at this folder
	git         gitState
	hideIgnored bool
}

type click struct {
	zone zone
	at   time.Time
}

// New builds the model on deps.Start.
func New(d Deps) *Model {
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Watcher == nil {
		d.Watcher = watch.Noop{}
	}
	if d.Runner == nil {
		d.Runner = opener.ExecRunner{}
	}
	s := fsmodel.Sort{By: fsmodel.Column(d.Cfg.Sort.By), Descending: d.Cfg.Sort.Descending, FoldersFirst: d.Cfg.Sort.FoldersFirst}
	km, _ := keymap.New(d.Cfg.Keys) // the problems were already reported as config warnings
	m := &Model{
		deps: d, resolve: opener.New(d.Cfg), keys: km, mouse: d.Cfg.Mouse,
		hist: history.New(d.Start), sort: s, hidden: d.Cfg.ShowHidden,
		lastSel: map[string]string{}, nextImgID: 1000, fresh: map[string]bool{},
		hideIgnored:   d.Cfg.Git.HideIgnored,
		imagesAllowed: d.Cfg.Preview.Images == "kitty",
	}
	m.restyle()
	if d.Warning == "" && len(d.Cfg.Warnings) > 0 {
		d.Warning = d.Cfg.Warnings[0]
		m.deps.Warning = d.Warning
	}
	if d.Warning != "" {
		m.setError(errors.New(d.Warning))
	}
	if err := m.load(d.Start, ""); err != nil {
		m.setError(err)
	}
	m.follow()
	return m
}

// Init asks the terminal whether it can show images, unless the config decides.
func (m *Model) Init() tea.Cmd {
	if m.deps.Cfg.Preview.Images == "auto" {
		return tea.Batch(tea.Raw(termimg.QuerySequence()), waitEvent(m.deps.Watcher))
	}
	return waitEvent(m.deps.Watcher)
}

func (m *Model) dir() string { return m.hist.Current() }

func (m *Model) setStatus(s string)       { m.status, m.statusErr = s, false }
func (m *Model) setError(err error)       { m.status, m.statusErr = err.Error(), true }
func (m *Model) entries() []fsmodel.Entry { return m.listing.Entries }

func (m *Model) selected() (fsmodel.Entry, bool) {
	if m.sel < 0 || m.sel >= len(m.listing.Entries) {
		return fsmodel.Entry{}, false
	}
	return m.listing.Entries[m.sel], true
}

// load reads dir into the model, selecting the entry called name (or the remembered one, or the first).
func (m *Model) load(dir, name string) error {
	l, err := fsmodel.ReadDir(dir, fsmodel.ReadOptions{ShowHidden: m.hidden, Sort: m.sort})
	if err != nil {
		return err
	}
	m.listing = m.readGit(l)
	if name == "" {
		name = m.lastSel[dir]
	}
	m.sel, m.top = 0, 0
	for i, e := range l.Entries {
		if e.Name == name {
			m.sel = i
			break
		}
	}
	m.ensureVisible()
	return nil
}

// remember stores the current selection of the directory we are about to leave.
func (m *Model) remember() {
	if e, ok := m.selected(); ok {
		m.lastSel[m.dir()] = e.Name
	}
}

// enter navigates to dir as a new history step. On failure nothing changes and the error is shown.
func (m *Model) enter(dir, name string) tea.Cmd {
	prev := m.listing
	m.remember()
	if err := m.load(dir, name); err != nil {
		m.listing = prev
		m.setError(err)
		return nil
	}
	m.hist = m.hist.Visit(dir)
	m.fresh = map[string]bool{}
	m.follow()
	m.setStatus("")
	return m.refreshPreview()
}

// step moves through the history (back or forward) and reloads.
func (m *Model) step(next history.History) tea.Cmd {
	if next.Current() == m.dir() {
		return nil
	}
	prev := m.listing
	m.remember()
	if err := m.load(next.Current(), ""); err != nil {
		m.listing = prev
		m.setError(err)
		return nil
	}
	m.hist = next
	m.fresh = map[string]bool{}
	m.follow()
	m.setStatus("")
	return m.refreshPreview()
}

func (m *Model) parent() tea.Cmd {
	d := m.dir()
	p := filepath.Dir(d)
	if p == d {
		return nil
	}
	return m.enter(p, filepath.Base(d))
}

// reload re-reads the current directory keeping the selection by name.
func (m *Model) reload() tea.Cmd {
	name := ""
	if e, ok := m.selected(); ok {
		name = e.Name
	}
	if err := m.load(m.dir(), name); err != nil {
		m.setError(err)
		return nil
	}
	return m.refreshPreview()
}

func (m *Model) resort(s fsmodel.Sort) tea.Cmd {
	name := ""
	if e, ok := m.selected(); ok {
		name = e.Name
	}
	m.sort = s
	m.listing = m.listing.Sorted(s)
	m.sel = 0
	for i, e := range m.listing.Entries {
		if e.Name == name {
			m.sel = i
		}
	}
	m.ensureVisible()
	return nil
}

func (m *Model) setSelection(i int) tea.Cmd {
	n := len(m.listing.Entries)
	if n == 0 {
		return nil
	}
	m.sel = max(0, min(n-1, i))
	m.ensureVisible()
	return m.refreshPreview()
}

func (m *Model) listRows() int { return max(1, m.panelH()-3) }

func (m *Model) ensureVisible() {
	rows := m.listRows()
	if m.sel < m.top {
		m.top = m.sel
	}
	if m.sel >= m.top+rows {
		m.top = m.sel - rows + 1
	}
	m.top = max(0, min(m.top, max(0, len(m.listing.Entries)-rows)))
}

// openSelected opens the selected entry: a directory is entered, a file goes to its opener.
func (m *Model) openSelected() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		return nil
	}
	if e.IsDir {
		return m.enter(e.Path, "")
	}
	return m.run(m.resolve.Resolve(e.Path))
}

// run starts cmd: GUI openers directly, terminal openers by handing the terminal over.
func (m *Model) run(c opener.Command) tea.Cmd {
	if c.Mode == config.ModeTerminal {
		return tea.ExecProcess(exec.Command(c.Argv[0], c.Argv[1:]...), func(err error) tea.Msg { return execDoneMsg{err: err} })
	}
	if err := m.resolve.Open(m.deps.Runner, c); err != nil {
		m.setError(err)
		return nil
	}
	m.setStatus(fmt.Sprintf(i18n.T("Abierto con %s", "Opened with %s"), c.Name))
	return nil
}

type execDoneMsg struct{ err error }

// Update handles input and asynchronous results.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ensureVisible()
		return m, m.refreshPreview()
	case tea.KeyPressMsg:
		return m, m.onKey(msg)
	case tea.MouseClickMsg:
		return m, m.onClick(msg)
	case tea.MouseWheelMsg:
		return m, m.onWheel(msg)
	case uv.KittyGraphicsEvent:
		if termimg.IsSupportReply(msg) && m.deps.Cfg.Preview.Images != "off" {
			m.imagesAllowed = true
		}
	case dirChangedMsg:
		return m, m.onDirChanged(msg)
	case previewMsg:
		return m, m.onPreview(msg)
	case imageMsg:
		return m, m.onImage(msg)
	case execDoneMsg:
		if msg.err != nil {
			m.setError(msg.err)
		}
		return m, tea.ClearScreen
	}
	return m, nil
}

// restyle rebuilds the styles from the config and the environment.
func (m *Model) restyle() {
	m.styles = newStyles(look{Theme: m.deps.Cfg.Theme, NoColor: m.deps.NoColor, ScreenBackground: m.deps.Cfg.ScreenBackground})
}
