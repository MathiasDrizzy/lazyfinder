package herdr

import (
	"bufio"
	"encoding/json"
	"net"
	"path/filepath"
	"strings"
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
