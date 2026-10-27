package herdr

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeServer answers each request line with reply(method, params) on a Unix socket, like herdr.
func fakeServer(t *testing.T, reply func(method string, params json.RawMessage) string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "h.sock")
	if len(path) > 100 {
		path = filepath.Join("/tmp", "lf-"+strings.ReplaceAll(t.Name(), "/", "-")+".sock")
	}
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				var req struct {
					ID     string          `json:"id"`
					Method string          `json:"method"`
					Params json.RawMessage `json:"params"`
				}
				line, _ := bufio.NewReader(conn).ReadBytes('\n')
				if json.Unmarshal(line, &req) != nil {
					return
				}
				_, _ = conn.Write([]byte(`{"id":"` + req.ID + `",` + reply(req.Method, req.Params) + "}\n"))
			}()
		}
	}()
	return path
}

func TestClientDecodesTheResultAndTheError(t *testing.T) {
	sock := fakeServer(t, func(method string, _ json.RawMessage) string {
		if method == "pane.list" {
			return `"result":{"type":"pane_list","panes":[{"pane_id":"w1:p1","tab_id":"w1:t1","focused":true,"cwd":"/x","label":"lazyfinder"}]}`
		}
		return `"error":{"code":"pane_not_found","message":"pane not found"}`
	})
	c := Client{Socket: sock}
	panes, err := c.Panes()
	if err != nil || len(panes) != 1 || panes[0].Label != Label || panes[0].Cwd != "/x" {
		t.Fatalf("%v %+v", err, panes)
	}
	err = c.Call("pane.close", map[string]string{"pane_id": "w9:p9"}, nil)
	if err == nil || !strings.Contains(err.Error(), "pane not found") {
		t.Fatalf("err = %v", err)
	}
}

// The sidebar asks herdr for the folder of the pane next to it every half second, in every tab. That is one pane.list
// and nothing else: telling which panes are sidebars takes a pane.process_info for each of them, and only the hooks need
// that.
func TestFollowingAFolderAsksHerdrForTheListOnly(t *testing.T) {
	var mu sync.Mutex
	var methods []string
	c := Client{Socket: fakeServer(t, func(method string, _ json.RawMessage) string {
		mu.Lock()
		defer mu.Unlock()
		methods = append(methods, method)
		return `"result":{"panes":[{"pane_id":"w1:p1","tab_id":"w1:t1","focused":true,"cwd":"/w/api"},{"pane_id":"w1:p2","tab_id":"w1:t1","label":"lazyfinder"}]}`
	})}
	panes, err := c.List()
	if err != nil || len(panes) != 2 {
		t.Fatalf("%v %v", panes, err)
	}
	if dir := FollowDir(panes, "w1:p2"); dir != "/w/api" {
		t.Fatalf("dir %q", dir)
	}
	if _, err := tabOf(c, "w1:p2"); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(methods, ","); got != "pane.list,pane.list" {
		t.Fatalf("calls: %s, want only pane.list twice", got)
	}
}

// The hooks do need to know the sidebars, and Panes says it.
func TestPanesStillSaysWhichAreSidebars(t *testing.T) {
	f := newFake()
	f.panes = append(f.panes, Pane{ID: "w1:p9", Tab: "w1:t1", Label: Label})
	panes, err := f.serve(t).Panes()
	if err != nil || len(panes) != 2 || panes[0].Sidebar || !panes[1].Sidebar {
		t.Fatalf("%+v %v", panes, err)
	}
}
