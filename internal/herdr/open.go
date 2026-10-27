package herdr

import (
	"errors"
	"strings"
)

// Open runs a terminal program in a new tab of herdr, in dir and with the focus, titled label from the first moment.
// The program is the command of the pane (layout.apply takes an argv), so no shell starts: the shell of the user reads
// its startup files first, which took a quarter of a second, and typing "exec ..." into it showed that line as the title
// of the tab. Quitting the program ends the pane, and the tab is marked so the hooks close it when only its sidebar is
// left. A herdr that answers layout.apply with an error gets the old way: a shell that hands itself over (exec).
func Open(c Client, e Env, dir string, argv []string, label string) error {
	if len(argv) == 0 {
		return errors.New("herdr: no program to open")
	}
	params := map[string]any{
		"focus":     true,
		"tab_label": label,
		"root":      map[string]any{"type": "pane", "label": label, "cwd": dir, "command": argv},
	}
	if e.WorkspaceID != "" {
		params["workspace_id"] = e.WorkspaceID
	}
	var r struct {
		Layout struct {
			Tab string `json:"tab_id"`
		} `json:"layout"`
	}
	err := c.Call("layout.apply", params, &r)
	var refused *CallError
	if errors.As(err, &refused) {
		return openInShell(c, e, dir, argv, label)
	}
	if err != nil {
		return err
	}
	if r.Layout.Tab == "" {
		return errors.New("herdr: the new tab has no pane")
	}
	return mark(e.StateDir, opened, r.Layout.Tab, true)
}

// openInShell is Open for a herdr without layout.apply: a tab with the shell of the user, which hands itself over to
// the program.
func openInShell(c Client, e Env, dir string, argv []string, label string) error {
	var r struct {
		Root struct {
			ID  string `json:"pane_id"`
			Tab string `json:"tab_id"`
		} `json:"root_pane"`
	}
	if err := c.Call("tab.create", map[string]any{"cwd": dir, "focus": true, "label": label}, &r); err != nil {
		return err
	}
	if r.Root.ID == "" {
		return errors.New("herdr: the new tab has no pane")
	}
	if err := mark(e.StateDir, opened, r.Root.Tab, true); err != nil {
		return err
	}
	quoted := make([]string, len(argv))
	for i, a := range argv {
		quoted[i] = shellQuote(a)
	}
	return c.Call("pane.send_text", map[string]string{"pane_id": r.Root.ID, "text": "exec " + strings.Join(quoted, " ") + "\n"}, nil)
}

// shellQuote puts s in single quotes for sh, bash, zsh and fish: a quote inside closes them, is escaped and opens
// them again.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// closeFinished closes the tabs lazyfinder opened for a program once the program quit: only the sidebar is left.
func closeFinished(c Client, e Env, panes []Pane) error {
	count, sidebar := map[string]int{}, map[string]bool{}
	for _, p := range panes {
		count[p.Tab]++
		if p.Sidebar {
			sidebar[p.Tab] = true
		}
	}
	var errs []error
	for tab, n := range count {
		if n == 1 && sidebar[tab] && marked(e.StateDir, opened, tab) {
			errs = append(errs, c.Call("tab.close", map[string]string{"tab_id": tab}, nil))
		}
	}
	return errors.Join(errs...)
}
