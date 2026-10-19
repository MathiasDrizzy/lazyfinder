package keymap

import (
	"os"
	"strings"
	"testing"
)

// docs/keybindings.md names every action by the key it has in config.json: an action missing there cannot be
// rebound by someone reading the docs.
func TestTheKeybindingsDocNamesEveryAction(t *testing.T) {
	doc, err := os.ReadFile("../../docs/keybindings.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range Actions {
		if !strings.Contains(string(doc), "`"+string(a)+"`") {
			t.Errorf("docs/keybindings.md lacks the action %q", a)
		}
	}
}
