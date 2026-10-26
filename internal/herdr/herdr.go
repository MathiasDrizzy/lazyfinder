// Package herdr runs lazyfinder as a herdr plugin (herdr-plugin.toml at the root of the repository). The event hooks
// of the manifest call Ensure, which docks one lazyfinder sidebar at the configured edge of each tab without taking
// the focus; the toggle action calls Toggle; and the sidebar follows the folder of the pane next to it.
//
// herdr speaks newline-delimited JSON over a Unix socket: one request per line, the answer echoes its id
// (https://herdr.dev/docs/socket-api).
package herdr

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	// PluginID is the id of herdr-plugin.toml.
	PluginID = "lazyfinder"
	// Entrypoint is the id of the [[panes]] entry that runs the sidebar.
	Entrypoint = "sidebar"
	// Label is the title of that entry: herdr names the pane with it, and pane.list has no other way to tell a pane
	// of the plugin from the rest.
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
}

// titled is true when p carries the title of the sidebar and can be it. A pane that runs an agent never is: plugins
// that name panes after their folder give the pane of the agent working in the lazyfinder folder that same title.
func (p Pane) titled() bool { return p.Label == Label && p.Agent == "" }

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
		return fmt.Errorf("herdr %s: %s (%s)", method, resp.Error.Message, resp.Error.Code)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(resp.Result, result); err != nil {
		return fmt.Errorf("herdr %s: %w", method, err)
	}
	return nil
}

// Panes lists every pane of the server.
func (c Client) Panes() ([]Pane, error) {
	var r struct {
		Panes []Pane `json:"panes"`
	}
	err := c.Call("pane.list", map[string]any{}, &r)
	return r.Panes, err
}

// LayoutOf is the layout of the tab of a pane.
func (c Client) LayoutOf(pane string) (Layout, error) {
	var r struct {
		Layout Layout `json:"layout"`
	}
	err := c.Call("pane.layout", map[string]string{"pane_id": pane}, &r)
	return r.Layout, err
}

// Restored is true when pane runs, in front, the program herdr started in it and that program is not lazyfinder.
// herdr restores the pane of the plugin after a restart as a shell that keeps its title: that pane is not a sidebar.
// A program lazyfinder opened in its pane (an editor) is in front of lazyfinder, so that pane is not restored.
func (c Client) Restored(pane string) (bool, error) {
	var r struct {
		Info struct {
			Root  int `json:"shell_pid"`
			Front []struct {
				Name string `json:"name"`
				PID  int    `json:"pid"`
			} `json:"foreground_processes"`
		} `json:"process_info"`
	}
	if err := c.Call("pane.process_info", map[string]string{"pane_id": pane}, &r); err != nil {
		return false, err
	}
	for _, p := range r.Info.Front {
		if p.PID == r.Info.Root {
			return p.Name != Label, nil
		}
	}
	return false, nil
}
