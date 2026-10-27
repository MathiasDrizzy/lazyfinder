package opener

import (
	"reflect"
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

func cfg() config.Config {
	c, _ := config.Parse([]byte(`{
	  "version": 1,
	  "openers": {
	    "code": {"argv": ["code", "{file}"], "mode": "gui"},
	    "vim":  {"argv": ["vim", "{file}"], "mode": "terminal"}
	  },
	  "open_with": {"txt": "code", "md": "vim"}
	}`), "/home/u")
	return c
}

func TestResolveByExtension(t *testing.T) {
	r := New(cfg())
	cmd := r.Resolve("/d/Notes.TXT")
	if cmd.Name != "code" || cmd.Mode != config.ModeGUI {
		t.Fatalf("cmd = %+v", cmd)
	}
	if !reflect.DeepEqual(cmd.Argv, []string{"code", "/d/Notes.TXT"}) {
		t.Fatalf("argv = %v", cmd.Argv)
	}
}

func TestDefaultIsSystemOpen(t *testing.T) {
	useGOOS(t, "darwin") // the macOS commands; linux_test.go checks the Linux ones
	cmd := New(cfg()).Resolve("/d/photo.png")
	if !reflect.DeepEqual(cmd.Argv, []string{"open", "--", "/d/photo.png"}) || cmd.Name != "default" {
		t.Fatalf("cmd = %+v", cmd)
	}
}

func TestHostileNamesStayOneArgument(t *testing.T) {
	r := New(cfg())
	for _, n := range []string{"a b.txt", "a;rm -rf ~.txt", "$(touch x).txt", "`id`.txt", "q'uote\".txt", "new\nline.txt"} {
		cmd := r.Resolve("/d/" + n)
		if len(cmd.Argv) != 2 || cmd.Argv[1] != "/d/"+n {
			t.Errorf("%q split or altered: %v", n, cmd.Argv)
		}
	}
}

func TestDashPrefixedNamesCannotBeOptions(t *testing.T) {
	useGOOS(t, "darwin")                 // the macOS commands; linux_test.go checks the Linux ones
	cmd := New(cfg()).Resolve("-rf.txt") // relative path starting with a dash
	if cmd.Argv[1] != "./-rf.txt" {
		t.Fatalf("argv = %v", cmd.Argv)
	}
	def := New(cfg()).Resolve("-x.png")
	if def.Argv[1] != "--" || def.Argv[2] != "-x.png" {
		t.Fatalf("default argv = %v", def.Argv)
	}
}

func TestChoicesListsConfiguredOpenersAndDefault(t *testing.T) {
	old := AppExists
	t.Cleanup(func() { AppExists = old })
	AppExists = func(string) bool { return false }
	t.Setenv("PATH", t.TempDir()) // built-in terminal editors are listed only when installed
	names := []string{}
	for _, c := range New(cfg()).Choices("/d/a.txt") {
		names = append(names, c.Name)
	}
	if !reflect.DeepEqual(names, []string{"code", "vim", "default"}) {
		t.Fatalf("names = %v (extension match first is not required, configured sorted, default last)", names)
	}
}

type recorder struct{ calls [][]string }

func (r *recorder) Start(argv []string) error { r.calls = append(r.calls, argv); return nil }

func TestOpenUsesRunner(t *testing.T) {
	rec := &recorder{}
	r := New(cfg())
	if err := r.Open(rec, r.Resolve("/d/a.txt")); err != nil {
		t.Fatal(err)
	}
	if len(rec.calls) != 1 || rec.calls[0][0] != "code" {
		t.Fatalf("calls = %v", rec.calls)
	}
}

func TestOpenRejectsTerminalMode(t *testing.T) {
	// Terminal-mode openers need the UI to suspend the TUI; Open must refuse to start them directly.
	rec := &recorder{}
	r := New(cfg())
	if err := r.Open(rec, r.Resolve("/d/a.md")); err == nil || len(rec.calls) != 0 {
		t.Fatalf("terminal opener must not be started through Open: err=%v calls=%v", err, rec.calls)
	}
}

func TestExecRunnerMissingBinary(t *testing.T) {
	err := ExecRunner{}.Start([]string{"/definitely/not/a/binary", "x"})
	if err == nil {
		t.Fatal("want error")
	}
}

func TestQuickLookIsFixedArgvWithOneFileArgument(t *testing.T) {
	useGOOS(t, "darwin") // the macOS commands; linux_test.go checks the Linux ones
	c := QuickLook("/d/a b;$(x).png")
	if !reflect.DeepEqual(c.Argv, []string{"qlmanage", "-p", "/d/a b;$(x).png"}) {
		t.Fatalf("argv = %v", c.Argv)
	}
	if QuickLook("-x.png").Argv[2] != "./-x.png" {
		t.Fatal("dash-prefixed names must not become options")
	}
}

func TestChoicesAddsInstalledBuiltInsAfterTheConfiguredOnes(t *testing.T) {
	old := AppExists
	t.Cleanup(func() { AppExists = old })
	AppExists = func(app string) bool { return app == "TextEdit" }
	t.Setenv("PATH", t.TempDir()) // no terminal editor installed
	var names []string
	for _, c := range New(cfg()).Choices("/d/a.txt") {
		names = append(names, c.Name)
	}
	want := []string{"code", "vim", "textedit", "default"} // vim is the user's; preview and vscode are not installed
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("choices = %v, want %v", names, want)
	}
}

func TestABuiltInCanBeMappedToAnExtension(t *testing.T) {
	c, err := config.Parse([]byte(`{"version": 1, "open_with": {"txt": "textedit"}}`), "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	got := New(c).Resolve("/d/a.txt")
	if !reflect.DeepEqual(got.Argv, []string{"open", "-a", "TextEdit", "/d/a.txt"}) {
		t.Fatalf("argv = %v", got.Argv)
	}
}

type dirRunner struct {
	dir  string
	argv []string
}

func (d *dirRunner) Start(argv []string) error { d.argv = argv; return nil }
func (d *dirRunner) StartIn(dir string, argv []string) error {
	d.dir, d.argv = dir, argv
	return nil
}

func TestCustomSubstitutesFileAndDirAsWholeArguments(t *testing.T) {
	c := Custom("count", config.Opener{Argv: []string{"wc", "-l", config.FilePlaceholder, config.DirPlaceholder}, Mode: config.ModeGUI}, "-odd name.txt", "/work/dir")
	want := []string{"wc", "-l", "./-odd name.txt", "/work/dir"}
	if strings.Join(c.Argv, "|") != strings.Join(want, "|") || c.Dir != "/work/dir" || c.Name != "count" {
		t.Fatalf("command = %+v", c)
	}
}

func TestOpenRunsACommandInItsFolderWhenTheRunnerCan(t *testing.T) {
	r := &dirRunner{}
	c := Command{Name: "make", Mode: config.ModeGUI, Argv: []string{"make"}, Dir: "/work"}
	if err := (Resolver{}).Open(r, c); err != nil {
		t.Fatal(err)
	}
	if r.dir != "/work" || r.argv[0] != "make" {
		t.Fatalf("dir = %q argv = %v", r.dir, r.argv)
	}
}

// A terminal program is titled with the file it opens, so a command says which file that is.
func TestACommandKnowsTheFileItOpens(t *testing.T) {
	cfg := config.Defaults("/h")
	cfg.Openers = map[string]config.Opener{"micro": {Argv: []string{"micro", config.FilePlaceholder}, Mode: config.ModeTerminal}}
	cfg.OpenWith = map[string]string{"md": "micro"}
	c := New(cfg).Resolve("/w/notes.md")
	if c.File != "/w/notes.md" || c.Name != "micro" {
		t.Fatalf("%+v", c)
	}
	if got := Custom("lint", cfg.Openers["micro"], "/w/a.go", "/w"); got.File != "/w/a.go" {
		t.Fatalf("custom: %+v", got)
	}
}
