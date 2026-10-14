// Package ui is the Bubble Tea front end. It is the only package that imports the TUI framework
// (docs/adr/0002-architecture.md).
package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/history"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
	"github.com/MathiasDrizzy/lazyfinder/internal/preview"
	"github.com/MathiasDrizzy/lazyfinder/internal/search"
	"github.com/MathiasDrizzy/lazyfinder/internal/termimg"
	"github.com/MathiasDrizzy/lazyfinder/internal/watch"
	uv "github.com/charmbracelet/ultraviolet"
)

const (
	minWidth          = 60
	minHeight         = 16
	followMinWidth    = 20
	followMinHeight   = 8
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

	// Probe and Raster are the external tools previews may use for media facts and PDF pages (nil: none).
	// ContentFind lists files containing a text for the in: search filter (ripgrep); nil uses the native scan.
	ContentFind search.Finder
	// Index is the system index for the whole-system search scope (Spotlight); nil means there is none and the
	// search walks the home folder instead.
	Index search.Indexer

	Probe  preview.Probe
	Raster preview.Raster

	// Follow is the project sidebar mode (--follow): no sidebar, a tree of the start folder that stays on it.
	Follow  bool
	Warning string // shown in the status line at start (for example an invalid config)

	// ConfigPath is where the settings popup saves; empty means it only changes the running session.
	ConfigPath string

	// LookPath finds an executable (nil: exec.LookPath); tests use it to say whether lazygit is installed.
	LookPath func(string) (string, error)
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

	watchSet     map[string]bool // the folders the watcher follows
	projTab      projectTab      // the tab of the project sidebar mode that is showing
	savedColumns []string        // the window columns, kept while the project mode shows names only
	expanded     map[string]bool // folders opened in place with +; kept while you move around
	treeDirs     []string        // the expanded folders that are on screen now
	fresh        map[string]bool // names that appeared while we looked at this folder
	git          gitState
	hideIgnored  bool
	search       searchState
	goto_        gotoState

	layout   string // list or columns
	drag     dragState
	ctx      ctxState
	help     helpState
	pointerX int // column of the last click, for dragging a border
	hover    hoverState
	pointerY int // row of the last click, for dragging the border between the sidebar boxes
	icons    icons.Set
	mil      millerState
	filter   string // quick filter on the names of this folder; empty means none
	picker   pickerState
	recents  []string // visited folders, newest first
	saved    []savedSearch
	tags     map[string][]string // path -> tag names, kept in tags.json next to config.json

	tabs []tabState // empty until a second tab is opened
	cur  int

	ops    *fileops.Ops
	marked map[string]bool // rows marked to act on together; cleared when the folder changes
	clip   clipboard
	prompt promptState
	info   infoState
	scm    scmState
	diff   diffState
	gm     gitMenuState
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
	km, _ := keymap.New(keymap.WithMode(d.Cfg.Keys, d.Cfg.KeybindingMode)) // the problems were already reported as config warnings
	m := &Model{
		deps: d, resolve: opener.New(d.Cfg), keys: km, mouse: d.Cfg.Mouse,
		hist: history.New(d.Start), sort: s, hidden: d.Cfg.ShowHidden,
		lastSel: map[string]string{}, nextImgID: 1000, fresh: map[string]bool{}, watchSet: map[string]bool{}, marked: map[string]bool{}, ops: fileops.New(d.Cfg.Ops.TrashDir), expanded: map[string]bool{},
		hideIgnored:   d.Cfg.Git.HideIgnored,
		imagesAllowed: d.Cfg.Preview.Images == "kitty",
	}
	loadState(m.stateFile(recentsFile), &m.recents)
	m.recents = m.recents[:min(len(m.recents), maxRecents)]
	loadState(m.stateFile(searchesFile), &m.saved)
	loadState(m.stateFile(tagsFile), &m.tags)
	if m.tags == nil {
		m.tags = map[string][]string{}
	}
	m.layout = d.Cfg.Layout
	m.icons = icons.Set{Style: icons.Style(d.Cfg.Icons), Map: d.Cfg.IconMap}
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
	m.syncMiller()
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

func (m *Model) setStatus(s string)       { m.status, m.statusErr = oneLine(s), false }
func (m *Model) setError(err error)       { m.status, m.statusErr = oneLine(err.Error()), true }
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
	if dir != m.listing.Dir {
		clear(m.marked)
		m.filter = ""
	}
	l = m.filtered(l)
	m.scanGit(dir)
	m.listing = m.withChildren(l)
	m.retag()
	if name == "" {
		name = m.lastSel[dir]
	}
	m.sel, m.top = 0, 0
	for i, e := range m.listing.Entries {
		if e.Depth == 0 && e.Name == name {
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
	if m.search.mode != searchOff {
		if m.search.cancel != nil {
			m.search.cancel()
		}
		m.search = searchState{}
	}
	prev := m.listing
	m.remember()
	if err := m.load(dir, name); err != nil {
		m.listing = prev
		m.setError(err)
		return nil
	}
	m.hist = m.hist.Visit(dir)
	m.noteVisit(dir)
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
	if m.search.mode != searchOff {
		if m.search.cancel != nil {
			m.search.cancel()
		}
		m.search = searchState{}
	}
	prev := m.listing
	m.remember()
	if err := m.load(next.Current(), ""); err != nil {
		m.listing = prev
		m.setError(err)
		return nil
	}
	m.hist = next
	m.noteVisit(next.Current())
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

// reload re-reads the current directory keeping the selection by path.
func (m *Model) reload() tea.Cmd {
	keep := ""
	if e, ok := m.selected(); ok {
		keep = e.Path
	}
	if err := m.load(m.dir(), ""); err != nil {
		m.setError(err)
		return nil
	}
	m.selectPath(keep)
	m.follow()
	return m.refreshPreview()
}

// selectPath moves the selection to the row of path, if it is on screen.
func (m *Model) selectPath(path string) {
	for i, e := range m.listing.Entries {
		if e.Path == path {
			m.sel = i
			break
		}
	}
	m.ensureVisible()
}

func (m *Model) resort(s fsmodel.Sort) tea.Cmd {
	m.sort = s
	if m.searchOn() { // the list holds results: order them, do not read the folder
		keep := ""
		if e, ok := m.selected(); ok {
			keep = e.Path
		}
		m.listing = m.listing.Sorted(s)
		m.sel = 0
		m.selectPath(keep)
		return nil
	}
	m.reload()
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

// rowH is how many lines a row of the detail list takes: two in the comfortable (wide) density, like Finder's roomy rows.
func (m *Model) rowH() int {
	if m.deps.Cfg.Density == config.DensityComfortable && !m.columnsOn() && !m.projectMode() {
		return 2
	}
	return 1
}

func (m *Model) listRows() int { return max(1, (m.panelH()-3)/m.rowH()) }

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
		if m.deps.Follow {
			return m.toggle(e.Path)
		}
		return m.enter(e.Path, "")
	}
	m.noteVisit(e.Path)
	return m.run(m.resolve.Resolve(e.Path))
}

// run starts cmd: GUI openers directly, terminal openers by handing the terminal over.
func (m *Model) run(c opener.Command) tea.Cmd {
	if c.Mode == config.ModeTerminal {
		cmd := exec.Command(c.Argv[0], c.Argv[1:]...)
		cmd.Dir = c.Dir
		return tea.ExecProcess(cmd, func(err error) tea.Msg { return execDoneMsg{err: err} })
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
	next, cmd := m.update(msg)
	m.syncMiller()
	return next, cmd
}

func (m *Model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.ensureVisible()
		return m, m.refreshPreview()
	case tea.KeyPressMsg:
		m.hover.on = false // the keyboard takes over from the pointer
		return m, m.onKey(msg)
	case tea.MouseClickMsg:
		return m, m.onClick(msg)
	case tea.MouseWheelMsg:
		return m, m.onWheel(msg)
	case tea.MouseMotionMsg:
		return m, m.onMotion(msg)
	case tea.MouseReleaseMsg:
		return m, m.onRelease(msg)
	case uv.KittyGraphicsEvent:
		if termimg.IsSupportReply(msg) && m.deps.Cfg.Preview.Images != "off" {
			m.imagesAllowed = true
		}
	case searchMsg:
		return m, m.onSearch(msg)
	case dirChangedMsg:
		return m, m.onDirChanged(msg)
	case previewMsg:
		return m, m.onPreview(msg)
	case imageMsg:
		return m, m.onImage(msg)
	case gitDoneMsg:
		return m, m.onGitDone(msg)
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

// tooSmallNow says whether the window is below the minimum for the current mode.
func (m *Model) tooSmallNow() bool {
	if m.deps.Follow {
		return m.w < followMinWidth || m.h < followMinHeight
	}
	return m.w < minWidth || m.h < minHeight
}

// oneLine makes text from outside (a git error, a file name) safe for the bottom row: a tab or a control code
// would move the cursor and push the row past the edge of the terminal.
func oneLine(s string) string {
	return strings.Join(strings.Fields(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)), " ")
}
