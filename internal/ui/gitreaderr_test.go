package ui

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// scanGit dropped the error of git.Read: with a git that cannot list the repository's settings (older than 2.26),
// every repository lost its marks, branch and Git tab without a word. The status line says why.
func TestAGitThatCannotBeReadSaysSo(t *testing.T) {
	e := newEnv(t)
	cleanRepo(e)
	real, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	bin := t.TempDir()
	script := "#!/bin/sh\ncase \" $* \" in *\" config \"*) exit 129;; esac\nexec " + real + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := e.m.load(e.root, ""); err != nil {
		t.Fatal(err)
	}
	if e.m.git.ok || !e.m.statusErr || e.m.status == "" {
		t.Fatalf("git ok %v, status %q (error %v): the failure must show", e.m.git.ok, e.m.status, e.m.statusErr)
	}
}
