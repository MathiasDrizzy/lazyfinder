package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, env map[string]string, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, func(k string) string { return env[k] }, &o, &e, func(Options) error { return nil })
	return code, o.String(), e.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := runCLI(t, nil, "--version")
	if code != 0 || !strings.HasPrefix(out, "lazyfinder ") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestPrintDefaultConfigIsValidJSONWithVersion(t *testing.T) {
	code, out, _ := runCLI(t, map[string]string{"HOME": "/h"}, "--print-default-config")
	if code != 0 || !strings.Contains(out, `"version": 1`) || !strings.Contains(out, "/h/Documents") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "--nope")
	if code != 2 || errOut == "" {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestStartDirMustExist(t *testing.T) {
	code, _, errOut := runCLI(t, map[string]string{"HOME": t.TempDir()}, filepath.Join(t.TempDir(), "missing"))
	if code != 1 || !strings.Contains(errOut, "missing") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestInvalidConfigStartsWithDefaultsAndWarns(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "c.json")
	_ = os.WriteFile(cfg, []byte(`{"version":1,"colums":[]}`), 0o600)
	var got Options
	var o, e bytes.Buffer
	env := map[string]string{"HOME": dir, "LAZYFINDER_CONFIG": cfg}
	code := run([]string{dir}, func(k string) string { return env[k] }, &o, &e, func(opt Options) error { got = opt; return nil })
	if code != 0 {
		t.Fatalf("code = %d, err=%q", code, e.String())
	}
	if !strings.Contains(e.String(), "colums") {
		t.Fatalf("warning should name the bad field: %q", e.String())
	}
	if got.Warning == "" || len(got.Cfg.Columns) != 5 {
		t.Fatalf("defaults not used: %+v", got)
	}
}

func TestStartDirDefaultsToCwd(t *testing.T) {
	var got Options
	var o, e bytes.Buffer
	env := map[string]string{"HOME": t.TempDir()}
	code := run(nil, func(k string) string { return env[k] }, &o, &e, func(opt Options) error { got = opt; return nil })
	wd, _ := os.Getwd()
	if code != 0 || got.Start != wd {
		t.Fatalf("code=%d start=%q want %q", code, got.Start, wd)
	}
}
