package main

import (
	"bytes"
	"strings"
	"testing"
)

// Order 2026-10-15-16: lazyfinder is a herdr plugin. herdr-plugin.toml calls --herdr ensure from its hooks and
// --herdr toggle from its action; the pane of the plugin starts in the sidebar mode and follows the folder.

func TestHerdrRejectsAnUnknownAction(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "--herdr", "dance")
	if code != exitUsage || !strings.Contains(errOut, "startup") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestHerdrOutsideHerdrSaysSo(t *testing.T) {
	code, _, errOut := runCLI(t, map[string]string{"HOME": t.TempDir()}, "--herdr", "ensure")
	if code != exitRuntime || !strings.Contains(errOut, "HERDR_SOCKET_PATH") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestThePaneOfThePluginStartsInTheSidebarAndFollowsHerdr(t *testing.T) {
	cases := []struct {
		env    map[string]string
		follow bool
	}{
		{map[string]string{"HERDR_PLUGIN_ID": "lazyfinder", "HERDR_SOCKET_PATH": "/tmp/h.sock", "HERDR_PANE_ID": "w1:p2"}, true},
		{map[string]string{"HERDR_PLUGIN_ID": "other", "HERDR_SOCKET_PATH": "/tmp/h.sock", "HERDR_PANE_ID": "w1:p2"}, false},
		{map[string]string{"HERDR_SOCKET_PATH": "/tmp/h.sock", "HERDR_PANE_ID": "w1:p2"}, false}, // a shell inside herdr
	}
	for _, c := range cases {
		c.env["HOME"] = t.TempDir()
		var got Options
		var o, e bytes.Buffer
		run([]string{c.env["HOME"]}, func(k string) string { return c.env[k] }, &o, &e, func(opts Options) (string, error) { got = opts; return "", nil })
		if got.Follow != c.follow || (got.HerdrDir != nil) != c.follow || (got.HerdrHide != nil) != c.follow {
			t.Errorf("%v: follow %v, herdr %v, hide %v", c.env, got.Follow, got.HerdrDir != nil, got.HerdrHide != nil)
		}
	}
}
