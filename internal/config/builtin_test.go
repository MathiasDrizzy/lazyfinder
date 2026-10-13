package config

import "testing"

func TestBuiltInOpenersAreAvailableWithoutConfiguring(t *testing.T) {
	c, err := Parse([]byte(`{"version": 1, "open_with": {"md": "vscode", "txt": "textedit"}}`), "/home/u")
	if err != nil {
		t.Fatalf("open_with may name a built-in opener: %v", err)
	}
	for _, name := range []string{"textedit", "preview", "vscode", "vim"} {
		if _, ok := c.Opener(name); !ok {
			t.Errorf("built-in opener %q is missing", name)
		}
	}
	if op, _ := c.Opener("vim"); op.Mode != ModeTerminal {
		t.Errorf("vim must run in the terminal, got %q", op.Mode)
	}
}

func TestAUserOpenerReplacesABuiltInOfTheSameName(t *testing.T) {
	c, err := Parse([]byte(`{"version": 1, "openers": {"vscode": {"argv": ["code", "{file}"], "mode": "gui"}}}`), "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if op, _ := c.Opener("vscode"); op.Argv[0] != "code" {
		t.Fatalf("argv = %v, want the user's code", op.Argv)
	}
}

func TestBuiltInsAreNotWrittenToTheDefaultConfigFile(t *testing.T) {
	if got := string(DefaultJSON("/home/u")); containsAny(got, "textedit", "TextEdit") {
		t.Fatalf("the default file must stay short, got:\n%s", got)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}
