package git

import (
	"os"
	"path/filepath"
	"testing"
)

// git status runs a status inside each submodule, which reads the submodule's own config (.git/modules/<name>/config)
// that readOnly never looked at: a folder from a zip whose submodule set core.fsmonitor ran its command as soon as
// lazyfinder showed the folder. Reading the status runs nothing from a submodule either.
func TestReadingARepositoryRunsNothingFromASubmoduleConfig(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	inner := fixture(t)
	top := fixture(t)
	marks := t.TempDir()
	run(t, top, "-c", "protocol.file.allow=always", "submodule", "add", "-q", inner, "mod")
	run(t, top, "commit", "-q", "-m", "add mod")
	put(t, filepath.Join(top, "mod", "kept.txt"), "changed\n") // a dirty submodule, so git looks inside
	run(t, filepath.Join(top, "mod"), "config", "core.fsmonitor", "touch "+filepath.Join(marks, "fsmonitor")+"; false")
	if _, ok, err := Read(top); err != nil || !ok {
		t.Fatalf("Read = %v, %v", ok, err)
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("reading the repository ran commands from a submodule config: %v", got)
	}
	run(t, filepath.Join(top, "mod"), "commit", "-q", "-am", "move the submodule on")
	put(t, filepath.Join(top, "mod", "kept.txt"), "dirty again\n")
	if err := os.Remove(filepath.Join(marks, "fsmonitor")); err != nil { // the commit above ran it, as it should
		t.Fatal(err)
	}
	if _, err := Diff(top, Change{Path: "mod", Code: Modified}, false); err != nil {
		t.Fatal(err)
	}
	if got := ran(t, marks); len(got) > 0 {
		t.Fatalf("the diff of a submodule ran commands from its config: %v", got)
	}
}
