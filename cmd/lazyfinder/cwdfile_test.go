package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Requirement (2026-10-14): quitting leaves the terminal in the folder lazyfinder was in, and a setting turns that
// off. A program cannot change the folder of the shell that started it, so lazyfinder
// writes the folder it ends in to the file of --cwd-file and a shell function (--shell-init) moves there, as yazi
// and lf do.

func runWithLastDir(t *testing.T, env map[string]string, last string, args ...string) (int, string) {
	t.Helper()
	var o, e bytes.Buffer
	code := run(args, func(k string) string { return env[k] }, &o, &e, func(Options) (string, error) { return last, nil })
	return code, e.String()
}

func TestTheCwdFileGetsTheFolderTheAppEndsIn(t *testing.T) {
	home, last := t.TempDir(), t.TempDir()
	file := filepath.Join(t.TempDir(), "cwd")
	code, errOut := runWithLastDir(t, map[string]string{"HOME": home}, last, "--cwd-file", file, home)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	got, err := os.ReadFile(file)
	if err != nil || string(got) != last {
		t.Fatalf("the cwd file holds %q (%v), want %q", got, err, last)
	}
}

func TestTheCwdFileStaysEmptyWhenCdOnExitIsOff(t *testing.T) {
	home, last := t.TempDir(), t.TempDir()
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(cfg, []byte(`{"version":1,"cd_on_exit":false}`), 0o600); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "cwd")
	code, errOut := runWithLastDir(t, map[string]string{"HOME": home, "LAZYFINDER_CONFIG": cfg}, last, "--cwd-file", file, home)
	if code != 0 {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
	if got, _ := os.ReadFile(file); len(got) != 0 {
		t.Fatalf("with cd_on_exit off the cwd file must stay empty, got %q", got)
	}
}

func TestShellInitRejectsAnUnknownShell(t *testing.T) {
	code, _, errOut := runCLI(t, nil, "--shell-init", "tcsh")
	if code != 2 || !strings.Contains(errOut, "tcsh") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

// The function of --shell-init runs in a real shell with a fake lazyfinder that ends in another folder: after it the
// shell is in that folder.
func TestTheShellFunctionMovesTheShellToTheLastFolder(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		t.Run(shell, func(t *testing.T) {
			bin, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s is not installed", shell)
			}
			code, script, errOut := runCLI(t, nil, "--shell-init", shell)
			if code != 0 {
				t.Fatalf("code=%d err=%q", code, errOut)
			}
			dir := t.TempDir()
			target, _ := filepath.EvalSymlinks(t.TempDir())
			fake := filepath.Join(dir, "lazyfinder")
			// the fake writes the target to the file after --cwd-file, as the real one does on exit
			fakeScript := "#!/bin/sh\n[ \"$1\" = --cwd-file ] && printf %s '" + target + "' > \"$2\"\n"
			if err := os.WriteFile(fake, []byte(fakeScript), 0o755); err != nil {
				t.Fatal(err)
			}
			initFile := filepath.Join(dir, "init."+shell)
			if err := os.WriteFile(initFile, []byte(script), 0o600); err != nil {
				t.Fatal(err)
			}
			var cmdline string
			if shell == "fish" {
				cmdline = "source " + initFile + "; lazyfinder; pwd -P"
			} else {
				cmdline = ". " + initFile + "; lazyfinder; pwd -P"
			}
			args := []string{"-c", cmdline}
			if shell == "fish" {
				args = []string{"--no-config", "-c", cmdline}
			}
			cmd := exec.Command(bin, args...)
			cmd.Dir = t.TempDir()
			cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin", "HOME=" + t.TempDir(), "TMPDIR=" + t.TempDir()}
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s: %v\n%s", shell, err, out)
			}
			if got := strings.TrimSpace(string(out)); got != target {
				t.Fatalf("%s ends in %q, want %q", shell, got, target)
			}
		})
	}
}

// ORDEN 2026-10-14-15: started without --cwd-file (no shell function), the UI gets the line for the user's shell;
// with --cwd-file it gets nothing.
func TestTheUIGetsTheShellLineOnlyWithoutTheShellFunction(t *testing.T) {
	cases := []struct {
		shell, file, line string
		withCwdFile       bool
	}{
		{"/bin/zsh", "~/.zshrc", `eval "$(lazyfinder --shell-init zsh)"`, false},
		{"/usr/local/bin/bash", "~/.bashrc", `eval "$(lazyfinder --shell-init bash)"`, false},
		{"/opt/homebrew/bin/fish", "~/.config/fish/config.fish", "lazyfinder --shell-init fish | source", false},
		{"/bin/tcsh", "~/.zshrc", `eval "$(lazyfinder --shell-init zsh)"`, false},
		{"/bin/zsh", "", "", true},
	}
	for _, c := range cases {
		home := t.TempDir()
		args := []string{home}
		if c.withCwdFile {
			args = []string{"--cwd-file", filepath.Join(t.TempDir(), "cwd"), home}
		}
		var got Options
		var o, e bytes.Buffer
		env := map[string]string{"HOME": home, "SHELL": c.shell}
		run(args, func(k string) string { return env[k] }, &o, &e, func(opts Options) (string, error) { got = opts; return "", nil })
		if got.ShellSetup.File != c.file || got.ShellSetup.Line != c.line {
			t.Errorf("%s (cwd-file %v): got %+v, want %q %q", c.shell, c.withCwdFile, got.ShellSetup, c.file, c.line)
		}
	}
}
