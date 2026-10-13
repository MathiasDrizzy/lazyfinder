package keymap

import (
	"strings"
	"testing"
)

func TestDefaultsBindEveryActionAndNoKeyTwice(t *testing.T) {
	seen := map[string]Action{}
	for _, a := range Actions {
		keys := Defaults[a]
		if len(keys) == 0 {
			t.Errorf("action %q has no default key", a)
		}
		for _, k := range keys {
			if prev, dup := seen[k]; dup {
				t.Errorf("key %q is bound to %q and %q by default", k, prev, a)
			}
			seen[k] = a
		}
	}
	if len(Defaults) != len(Actions) {
		t.Errorf("Defaults has %d actions, Actions lists %d", len(Defaults), len(Actions))
	}
}

func TestNewWithoutOverridesUsesDefaults(t *testing.T) {
	m, warns := New(nil)
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if a, ok := m.Lookup("j"); !ok || a != Down {
		t.Fatalf("j should be down, got %q %v", a, ok)
	}
	if got := m.Keys(Quit); len(got) != 1 || got[0] != "q" {
		t.Fatalf("quit keys = %v", got)
	}
}

func TestOverrideReplacesTheKeysOfOneActionOnly(t *testing.T) {
	m, warns := New(map[string][]string{"quit": {"Z", "ctrl+q"}})
	if len(warns) != 0 {
		t.Fatalf("unexpected warnings: %v", warns)
	}
	if _, ok := m.Lookup("q"); ok {
		t.Error("q must stop quitting once quit is rebound")
	}
	if a, _ := m.Lookup("Z"); a != Quit {
		t.Errorf("x should quit, got %q", a)
	}
	if a, _ := m.Lookup("j"); a != Down {
		t.Errorf("other actions keep their keys, j = %q", a)
	}
	if got := m.Keys(Quit); strings.Join(got, ",") != "Z,ctrl+q" {
		t.Errorf("Keys(Quit) = %v", got)
	}
}

func TestUnknownActionIsAWarningAndIgnored(t *testing.T) {
	m, warns := New(map[string][]string{"teleport": {"t"}})
	if len(warns) != 1 || !strings.Contains(warns[0], `"teleport"`) {
		t.Fatalf("want one warning naming the action, got %v", warns)
	}
	if _, ok := m.Lookup("t"); ok {
		t.Error("an unknown action must bind nothing")
	}
}

func TestConflictKeepsTheDefaultOfTheOffendingAction(t *testing.T) {
	// "j" is down by default; binding it to up conflicts with the (unchanged) down action.
	m, warns := New(map[string][]string{"up": {"j"}})
	if len(warns) != 1 || !strings.Contains(warns[0], `"j"`) {
		t.Fatalf("want one warning naming the key, got %v", warns)
	}
	if a, _ := m.Lookup("j"); a != Down {
		t.Errorf("j must stay down, got %q", a)
	}
	if got := m.Keys(Up); strings.Join(got, ",") != strings.Join(Defaults[Up], ",") {
		t.Errorf("up must keep its defaults, got %v", got)
	}
}

func TestRebindingTwoActionsToTheSameKeyIsAConflict(t *testing.T) {
	_, warns := New(map[string][]string{"sort": {"z"}, "reload": {"z"}})
	if len(warns) != 1 {
		t.Fatalf("want one warning, got %v", warns)
	}
}

func TestEmptyOrBlankKeyIsRejected(t *testing.T) {
	_, warns := New(map[string][]string{"quit": {""}})
	if len(warns) != 1 {
		t.Fatalf("want one warning, got %v", warns)
	}
	_, warns = New(map[string][]string{"quit": {}})
	if len(warns) != 1 {
		t.Fatalf("an empty list is not a binding: %v", warns)
	}
}

func TestSwappingTwoActionsIsNotAConflict(t *testing.T) {
	m, warns := New(map[string][]string{"up": {"j"}, "down": {"k"}})
	if len(warns) != 0 {
		t.Fatalf("swapping is legal: %v", warns)
	}
	if a, _ := m.Lookup("j"); a != Up {
		t.Errorf("j = %q, want up", a)
	}
}
