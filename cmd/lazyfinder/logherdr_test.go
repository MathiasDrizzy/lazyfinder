package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/herdr"
)

// The log leaves file names out unless --log-paths asks for them, and is readable by its owner only.
func TestTheLogKeepsNamesOutUnlessAsked(t *testing.T) {
	for _, withPaths := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "lf.log")
		log, closeLog, err := newLogger(path, withPaths, nil)
		if err != nil {
			t.Fatal(err)
		}
		log.Info("opened", "path", "/secret/plan.txt", "name", "plan.txt", "size", 3)
		closeLog()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "plan.txt") != withPaths || !strings.Contains(string(data), `"size":3`) {
			t.Fatalf("withPaths=%v: %s", withPaths, data)
		}
		if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
			t.Fatalf("log mode %v", fi.Mode().Perm())
		}
	}
	if _, _, err := newLogger(filepath.Join(t.TempDir(), "missing", "lf.log"), false, nil); err == nil {
		t.Fatal("a log in a missing folder must fail")
	}
}

func TestHerdrStepsFailClearlyOutsideHerdr(t *testing.T) {
	var errOut bytes.Buffer
	if code := runHerdr("dance", func(string) string { return "" }, t.TempDir(), &errOut); code != exitUsage {
		t.Fatalf("unknown action: code %d", code)
	}
	errOut.Reset()
	if code := runHerdr("toggle", func(string) string { return "" }, t.TempDir(), &errOut); code != exitRuntime || !strings.Contains(errOut.String(), "HERDR_SOCKET_PATH") {
		t.Fatalf("no socket: code %d, %q", code, errOut.String())
	}
	env := map[string]string{"HERDR_SOCKET_PATH": filepath.Join(t.TempDir(), "none.sock"), "LAZYFINDER_CONFIG": filepath.Join(t.TempDir(), "c.json")}
	for _, action := range []string{"ensure", "toggle"} {
		errOut.Reset()
		if code := runHerdr(action, func(k string) string { return env[k] }, t.TempDir(), &errOut); code != exitRuntime {
			t.Fatalf("%s with a dead socket: code %d, %q", action, code, errOut.String())
		}
	}
}

func TestTheHerdrFollowerOnlyRunsAsThePluginPane(t *testing.T) {
	env := map[string]string{"HERDR_SOCKET_PATH": filepath.Join(t.TempDir(), "none.sock"), "HERDR_PANE_ID": "p1"}
	if herdrFollower(func(k string) string { return env[k] }) != nil {
		t.Fatal("outside the plugin there is nothing to follow")
	}
	env["HERDR_PLUGIN_ID"] = herdr.PluginID
	follow := herdrFollower(func(k string) string { return env[k] })
	if follow == nil || follow() != "" {
		t.Fatal("with herdr unreachable the follower keeps the folder shown")
	}
}
