// Package herdr runs lazyfinder as a herdr plugin (herdr-plugin.toml at the root of the repository). The event hooks
// of the manifest call Ensure, which docks one lazyfinder sidebar at the configured edge of each tab without taking
// the focus; the toggle action calls Toggle; and the sidebar follows the folder of the pane next to it.
//
// herdr speaks newline-delimited JSON over a Unix socket: one request per line, the answer echoes its id
// (https://herdr.dev/docs/socket-api).
package herdr

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// PluginID is the id of herdr-plugin.toml.
	PluginID = "lazyfinder"
	// Entrypoint is the id of the [[panes]] entry that runs the sidebar.
	Entrypoint = "sidebar"
	// Label is the title of that entry. It only marks a pane herdr restored after a restart as a shell: the sidebar is
	// told by the program it runs (Pane.Sidebar), since plugins that name panes after their folder reuse the title.
	Label = "lazyfinder"

	callTimeout = 2 * time.Second
)

// Pane is one pane of pane.list.
type Pane struct {
	ID            string `json:"pane_id"`
	Tab           string `json:"tab_id"`
	Workspace     string `json:"workspace_id"`
	Focused       bool   `json:"focused"`
	Cwd           string `json:"cwd"`
	ForegroundCwd string `json:"foreground_cwd"`
	Label         string `json:"label"`
	Agent         string `json:"agent"` // the coding agent running in the pane, if herdr detected one
	Sidebar       bool   `json:"-"`
	Stale         bool   `json:"-"`
}

// Sidebar is true when the pane runs lazyfinder as its own program (the root process of the pane), whatever title the
// pane has: plugins that name panes after their folder rename them. Stale is true for a pane herdr restored after a
// restart as an idle shell that kept the title of the sidebar. Both come from Client.Panes, not from pane.list.
//
// A pane that runs an agent is neither: the pane of the agent working in the lazyfinder folder can carry that same
// title, and the toggle must never close it.

// Dir is the folder the pane is in: the one of the program in front (a shell after cd), else the pane's own.
func (p Pane) Dir() string {
	if p.ForegroundCwd != "" {
		return p.ForegroundCwd
	}
	return p.Cwd
}

// Rect is a rectangle of pane.layout, in cells.
type Rect struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// LayoutPane is a pane of pane.layout.
type LayoutPane struct {
	ID   string `json:"pane_id"`
	Rect Rect   `json:"rect"`
}

// Layout is the tab of a pane as pane.layout gives it.
type Layout struct {
	Panes []LayoutPane `json:"panes"`
}

// Client calls herdr over its socket (HERDR_SOCKET_PATH).
type Client struct {
	Socket string
	// RootName tells which program a process runs; nil asks ps. A test sets it.
	RootName func(pid int) string
}

// CallError is the error herdr answered a request with: it understood nothing of it, so nothing was done.
type CallError struct{ Method, Message, Code string }

func (e *CallError) Error() string {
	return fmt.Sprintf("herdr %s: %s (%s)", e.Method, e.Message, e.Code)
}

// Call sends one request and decodes its result into result (nil ignores it).
func (c Client) Call(method string, params, result any) error {
	conn, err := net.DialTimeout("unix", c.Socket, callTimeout)
	if err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(callTimeout))
	req, err := json.Marshal(map[string]any{"id": "lazyfinder", "method": method, "params": params})
	if err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	if _, err := conn.Write(append(req, '\n')); err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	var resp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &resp); err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	if resp.Error != nil {
		return &CallError{Method: method, Message: resp.Error.Message, Code: resp.Error.Code}
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	return nil
}

// List lists every pane of the server, as herdr says it. The sidebar asks this every half second to follow the folder
// of the pane next to it, so it is one call and nothing more.
func (c Client) List() ([]Pane, error) {
	var r struct {
		Panes []Pane `json:"panes"`
	}
	err := c.Call("pane.list", map[string]any{}, &r)
	return r.Panes, err
}

// Panes is List that also says which panes are the sidebar, by the program they run: one pane.process_info for each
// pane that runs no agent. The hooks need it; the sidebar polling the folder does not.
func (c Client) Panes() ([]Pane, error) {
	panes, err := c.List()
	if err != nil {
		return nil, err
	}
	r := struct{ Panes []Pane }{panes}
	for i, p := range r.Panes {
		if p.Agent == "" {
			r.Panes[i].Sidebar, r.Panes[i].Stale = c.probe(p)
		}
	}
	return r.Panes, nil
}

// probe says whether p runs lazyfinder as its own program, and whether it is a sidebar herdr restored as an idle shell
// that kept the title. herdr restores the pane of the plugin after a restart as a shell with its title: that pane is not
// a sidebar, and only that one (an idle shell with the title) is closed. A pane that cannot be read is neither, so
// nothing is closed on a guess.
func (c Client) probe(p Pane) (sidebar, stale bool) {
	var r struct {
		Info struct {
			Root  int `json:"shell_pid"`
			Front []struct {
				Name string `json:"name"`
				PID  int    `json:"pid"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	}
	if err := c.Call("pane.process_info", map[string]string{"pane_id": p.ID}, &r); err != nil || r.Info.Root == 0 {
		return false, false
	}
	for _, f := range r.Info.Front {
		if f.PID == r.Info.Root { // the root process is in front: it is the program of the pane
			isMe := filepath.Base(f.Name) == Label
			return isMe, !isMe && p.Label == Label
		}
	}
	// Something the root started is in front (an editor opened from the sidebar, a command of the user): the root
	// process says what the pane is.
	return filepath.Base(c.rootName(r.Info.Root)) == Label, false
}

// rootName is the program the process pid runs.
func (c Client) rootName(pid int) string {
	if c.RootName != nil {
		return c.RootName(pid)
	}
	ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// LayoutOf is the layout of the tab of a pane.
func (c Client) LayoutOf(pane string) (Layout, error) {
	var r struct {
		Layout Layout `json:"layout"`
	}
	err := c.Call("pane.layout", map[string]string{"pane_id": pane}, &r)
	return r.Layout, err
}
