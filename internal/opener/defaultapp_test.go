package opener

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// The system row of Open With names the application macOS opens a file with; a name with quotes or a $ stays one
// argument of osascript.
func TestDefaultAppNamesTheApplicationOfTheSystem(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Launch Services is macOS")
	}
	p := filepath.Join(t.TempDir(), `a "b" $(x)'.txt`)
	if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := DefaultApp(p); got == "" {
		t.Fatal("no application for a .txt file")
	}
}

// With no application for the file, osascript prints nothing: that is no name, not ".".
func TestAnEmptyAnswerIsNoApplication(t *testing.T) {
	for out, want := range map[string]string{"": "", "\n": "", "/System/Applications/TextEdit.app\n": "TextEdit"} {
		if got := appName(out); got != want {
			t.Errorf("appName(%q) = %q, want %q", out, got, want)
		}
	}
}
