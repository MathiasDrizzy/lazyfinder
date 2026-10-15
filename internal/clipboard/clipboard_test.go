package clipboard

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// fakeTool writes a script named name into a new folder that saves its input to a log, and returns the folder.
func fakeTool(t *testing.T, name string) (dir, log string) {
	t.Helper()
	dir = t.TempDir()
	log = filepath.Join(dir, name+".log")
	script := "#!/bin/sh\nprintf '%s ' \"$*\" >>'" + log + "'\n/bin/cat >>'" + log + "'\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, log
}

func TestCopyPipesTheTextIntoTheFirstToolFound(t *testing.T) {
	dir, log := fakeTool(t, "xclip")
	t.Setenv("PATH", dir)
	if err := Copy("/tmp/some folder"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(log)
	if string(got) != "-selection clipboard /tmp/some folder" {
		t.Fatalf("xclip got %q", got)
	}
}

func TestPbcopyGoesBeforeTheOthers(t *testing.T) {
	dir, log := fakeTool(t, "pbcopy")
	if err := os.WriteFile(filepath.Join(dir, "xclip"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := Copy("/a"); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(log); string(got) != " /a" {
		t.Fatalf("pbcopy got %q", got)
	}
}

func TestWithoutAToolCopySaysThereIsNone(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := Copy("/a"); !errors.Is(err, ErrNoTool) {
		t.Fatalf("err %v, want ErrNoTool", err)
	}
}

func TestAToolThatFailsIsAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\necho no display >&2\nexit 3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	err := Copy("/a")
	var ee *exec.ExitError
	if err == nil || errors.Is(err, ErrNoTool) || !errors.As(err, &ee) {
		t.Fatalf("err %v, want the failure of pbcopy", err)
	}
}
