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
	code = run(args, func(k string) string { return env[k] }, &o, &e, func(Options) (string, error) { return "", nil })
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
	code := run([]string{dir}, func(k string) string { return env[k] }, &o, &e, func(opt Options) (string, error) { got = opt; return "", nil })
	if code != 0 {
		t.Fatalf("code = %d, err=%q", code, e.String())
	}
	if !strings.Contains(e.String(), "colums") {
		t.Fatalf("warning should name the bad field: %q", e.String())
	}
	if len(got.Cfg.Warnings) != 1 || len(got.Cfg.Columns) != 5 {
		t.Fatalf("defaults not used: %+v", got)
	}
}

func TestStartDirDefaultsToCwd(t *testing.T) {
	var got Options
	var o, e bytes.Buffer
	env := map[string]string{"HOME": t.TempDir()}
	code := run(nil, func(k string) string { return env[k] }, &o, &e, func(opt Options) (string, error) { got = opt; return "", nil })
	wd, _ := os.Getwd()
	if code != 0 || got.Start != wd {
		t.Fatalf("code=%d start=%q want %q", code, got.Start, wd)
	}
}

func runOptions(t *testing.T, env map[string]string, args ...string) (int, Options, string) {
	t.Helper()
	var got Options
	var o, e bytes.Buffer
	code := run(args, func(k string) string { return env[k] }, &o, &e, func(opt Options) (string, error) { got = opt; return "", nil })
	return code, got, e.String()
}

func TestFollowStartsTheProjectSidebarMode(t *testing.T) {
	proj := t.TempDir()
	code, got, errOut := runOptions(t, map[string]string{"HOME": t.TempDir()}, "--follow", proj)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if !got.Follow || got.Start != proj {
		t.Fatalf("follow=%v start=%q, want true and %q", got.Follow, got.Start, proj)
	}
	if len(got.Cfg.Columns) != 1 || got.Cfg.Columns[0] != "name" {
		t.Fatalf("the narrow tree shows only names, got %v", got.Cfg.Columns)
	}
}

func TestFollowAndADirectoryTogetherAreAUsageError(t *testing.T) {
	code, _, errOut := runOptions(t, map[string]string{"HOME": t.TempDir()}, "--follow", t.TempDir(), t.TempDir())
	if code != 2 || !strings.Contains(errOut, "--follow") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestFollowNeedsAFolderThatExists(t *testing.T) {
	code, _, errOut := runOptions(t, map[string]string{"HOME": t.TempDir()}, "--follow", filepath.Join(t.TempDir(), "gone"))
	if code != 1 || !strings.Contains(errOut, "gone") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

func TestWithoutFollowTheFullViewIsUsed(t *testing.T) {
	_, got, _ := runOptions(t, map[string]string{"HOME": t.TempDir()}, t.TempDir())
	if got.Follow || len(got.Cfg.Columns) != 5 {
		t.Fatalf("follow=%v columns=%v", got.Follow, got.Cfg.Columns)
	}
}

func TestCompletionsPrintAScriptPerShell(t *testing.T) {
	for shell, want := range map[string]string{"bash": "complete -F", "zsh": "#compdef lazyfinder", "fish": "complete -c lazyfinder"} {
		code, out, _ := runCLI(t, nil, "--completions", shell)
		if code != 0 || !strings.Contains(out, want) {
			t.Fatalf("%s: code=%d out=%q", shell, code, out)
		}
		for _, flag := range []string{"follow", "theme", "lang", "no-mouse", "print-default-config", "version", "completions", "cwd-file", "shell-init"} {
			if !strings.Contains(out, flag) {
				t.Errorf("%s completion lacks --%s", shell, flag)
			}
		}
	}
}

func TestCompletionsRejectAnUnknownShell(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "--completions", "powershell")
	if code != exitUsage || !strings.Contains(errOut, "bash, zsh or fish") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

// helpFlags are the flags --help lists, so a new flag cannot be left out of the man page or the completions.
func helpFlags(t *testing.T) []string {
	t.Helper()
	_, out, errOut := runCLI(t, nil, "--help")
	var flags []string
	for line := range strings.Lines(out + errOut) {
		if f, ok := strings.CutPrefix(line, "  -"); ok {
			flags = append(flags, strings.Fields(f)[0])
		}
	}
	if len(flags) < 10 {
		t.Fatalf("--help lists %d flags:\n%s%s", len(flags), out, errOut)
	}
	return flags
}

func TestManPageMentionsEveryFlag(t *testing.T) {
	page, err := os.ReadFile("../../docs/lazyfinder.1")
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range helpFlags(t) {
		if !strings.Contains(strings.ReplaceAll(string(page), `\-`, "-"), "--"+flag) {
			t.Errorf("man page lacks --%s", flag)
		}
	}
}

func TestEveryCompletionOffersEveryFlag(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		_, script, _ := runCLI(t, nil, "--completions", shell)
		for _, flag := range helpFlags(t) {
			if !strings.Contains(script, flag) {
				t.Errorf("%s completion lacks --%s", shell, flag)
			}
		}
	}
}
