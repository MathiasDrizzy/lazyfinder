package herdr

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

// Env is what herdr gives a hook or an action of the plugin.
type Env struct {
	PluginID string // HERDR_PLUGIN_ID; PluginID when empty
	TabID    string // HERDR_TAB_ID
	Event    string // HERDR_PLUGIN_EVENT_JSON; empty for an action
	StateDir string // HERDR_PLUGIN_STATE_DIR: the lock and the tabs whose sidebar the user closed
}

// Settings are the herdr section of config.json.
type Settings struct {
	Dock     string // left or right
	AutoOpen bool
	Width    int // percent of the tab
}

// Ensure docks a sidebar in the tab of the event when it has none, the user did not close it there, and auto_open
// is on. It never takes the focus and never opens a second one.
func Ensure(c Client, e Env, s Settings) error {
	if !s.AutoOpen {
		return nil
	}
	unlock, ok := lock(e.StateDir)
	if !ok {
		return nil // another hook is docking: the events its own layout changes fire land here
	}
	defer unlock()
	panes, err := c.Panes()
	if err != nil {
		return err
	}
	forget(e.StateDir, panes)
	tab := EventTab(e.Event, panes, e.TabID)
	if tab == "" || snoozed(e.StateDir, tab) {
		return nil
	}
	if _, ok, err := sidebarOf(c, panes, tab); ok || err != nil {
		return err
	}
	_, err = open(c, e, s, panes, tab, false)
	return err
}

// Startup runs once herdr restored its session (the [[startup]] hook): every tab whose sidebar came back as a shell
// gets a sidebar again, without the focus, unless auto_open is off.
func Startup(c Client, e Env, s Settings) error {
	unlock, ok := lock(e.StateDir)
	if !ok {
		return nil
	}
	defer unlock()
	panes, err := c.Panes()
	if err != nil {
		return err
	}
	forget(e.StateDir, panes)
	var errs []error
	for _, tab := range restoredTabs(panes) {
		_, live, err := sidebarOf(c, panes, tab)
		if err != nil || live || !s.AutoOpen || snoozed(e.StateDir, tab) {
			errs = append(errs, err)
			continue
		}
		_, err = open(c, e, s, panes, tab, false)
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// restoredTabs are the tabs with a pane titled as the sidebar, in the order of pane.list.
func restoredTabs(panes []Pane) []string {
	var tabs []string
	for _, p := range panes {
		if p.Label == Label && !slices.Contains(tabs, p.Tab) {
			tabs = append(tabs, p.Tab)
		}
	}
	return tabs
}

// Hide keeps the tab of pane without a sidebar until the toggle opens it again: the user left the sidebar pane with q.
func Hide(c Client, e Env, pane string) error {
	panes, err := c.Panes()
	if err != nil {
		return err
	}
	for _, p := range panes {
		if p.ID == pane {
			return snooze(e.StateDir, p.Tab, true)
		}
	}
	return fmt.Errorf("herdr: no pane %s", pane)
}

// sidebarOf is the sidebar of a tab. A pane titled as the sidebar that herdr restored as a shell is closed on the way:
// it is not a sidebar, and it would keep the tab from getting one.
func sidebarOf(c Client, panes []Pane, tab string) (Pane, bool, error) {
	for _, p := range panes {
		if p.Tab != tab || p.Label != Label {
			continue
		}
		restored, err := c.Restored(p.ID)
		if err != nil {
			return Pane{}, false, err
		}
		if !restored {
			return p, true, nil
		}
		if err := c.Call("pane.close", map[string]string{"pane_id": p.ID}, nil); err != nil {
			return Pane{}, false, err
		}
	}
	return Pane{}, false, nil
}

// Toggle is the action of the plugin on the focused tab: it opens the sidebar (with the focus), focuses it when it
// is open elsewhere, and closes it when it has the focus. A sidebar closed this way stays closed in that tab until
// the next Toggle.
func Toggle(c Client, e Env, s Settings) error {
	unlock, ok := lock(e.StateDir)
	if !ok {
		return errors.New("herdr: another lazyfinder hook is running; try again")
	}
	defer unlock()
	panes, err := c.Panes()
	if err != nil {
		return err
	}
	tab := EventTab("", panes, e.TabID)
	if tab == "" {
		return errors.New("herdr: no tab is focused")
	}
	side, ok, err := sidebarOf(c, panes, tab)
	if err != nil {
		return err
	}
	switch {
	case ok && side.Focused:
		if err := snooze(e.StateDir, tab, true); err != nil {
			return err
		}
		return c.Call("pane.close", map[string]string{"pane_id": side.ID}, nil)
	case ok:
		return c.Call("pane.focus", map[string]string{"pane_id": side.ID}, nil)
	}
	if err := snooze(e.StateDir, tab, false); err != nil {
		return err
	}
	_, err = open(c, e, s, panes, tab, true)
	return err
}

// open splits the pane at the dock edge of the tab, puts the sidebar on that side at its width and gives the focus
// to the sidebar (focus) or back to the pane that had it. plugin.pane.open only splits to the right, so a left dock
// swaps the two panes after it.
func open(c Client, e Env, s Settings, panes []Pane, tab string, focus bool) (string, error) {
	anchor, ok := Anchor(panes, tab)
	if !ok {
		return "", nil
	}
	l, err := c.LayoutOf(anchor.ID)
	if err != nil {
		return "", err
	}
	edge := EdgePane(l, s.Dock)
	if edge == "" {
		edge = anchor.ID
	}
	plugin := e.PluginID
	if plugin == "" {
		plugin = PluginID
	}
	params := map[string]any{
		"plugin_id": plugin, "entrypoint": Entrypoint, "placement": "split",
		"target_pane_id": edge, "direction": "right", "focus": false,
	}
	if d := anchor.Dir(); d != "" {
		params["cwd"] = d
	}
	var opened struct {
		PluginPane struct {
			Pane Pane `json:"pane"`
		} `json:"plugin_pane"`
	}
	if err := c.Call("plugin.pane.open", params, &opened); err != nil {
		return "", err
	}
	id := opened.PluginPane.Pane.ID
	if id == "" {
		return "", errors.New("herdr plugin.pane.open: no pane id in the answer")
	}
	direction := "right"
	if s.Dock == "left" {
		direction = "left"
		if err := c.Call("pane.swap", map[string]string{"source_pane_id": id, "target_pane_id": edge}, nil); err != nil {
			return id, err
		}
	}
	if a := resizeAmount(s.Width); a != 0 {
		if err := c.Call("pane.resize", map[string]any{"pane_id": id, "direction": direction, "amount": a}, nil); err != nil {
			return id, err
		}
	}
	back := anchor.ID
	if focus {
		back = id
	}
	return id, c.Call("pane.focus", map[string]string{"pane_id": back}, nil)
}

// lock takes the lock of the plugin without waiting; ok is false when another process holds it.
func lock(dir string) (unlock func(), ok bool) {
	if dir == "" {
		dir = os.TempDir()
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return func() {}, true // no state folder: run unlocked rather than never
	}
	f, err := os.OpenFile(filepath.Join(dir, "lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}, true
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, false
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, true
}

// forget drops the marks of tabs that are gone, so a later tab never inherits one.
func forget(dir string, panes []Pane) {
	if dir == "" || len(panes) == 0 {
		return // no list to trust: keep every mark
	}
	entries, err := os.ReadDir(filepath.Join(dir, "snoozed"))
	if err != nil {
		return
	}
	live := map[string]bool{}
	for _, p := range panes {
		live[filepath.Base(snoozePath(dir, p.Tab))] = true
	}
	for _, en := range entries {
		if !live[en.Name()] {
			_ = os.Remove(filepath.Join(dir, "snoozed", en.Name())) // a mark that cannot go only stays
		}
	}
}

// snoozePath is the marker of a tab whose sidebar the user closed; the tab id is made a plain file name.
func snoozePath(dir, tab string) string {
	safe := strings.NewReplacer("/", "_", ":", "_", "\\", "_", ".", "_").Replace(tab)
	return filepath.Join(dir, "snoozed", safe)
}

func snoozed(dir, tab string) bool {
	_, err := os.Stat(snoozePath(dir, tab))
	return err == nil
}

// snooze marks (on) or clears the tab.
func snooze(dir, tab string, on bool) error {
	p := snoozePath(dir, tab)
	if !on {
		if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("herdr: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("herdr: %w", err)
	}
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		return fmt.Errorf("herdr: %w", err)
	}
	return nil
}
