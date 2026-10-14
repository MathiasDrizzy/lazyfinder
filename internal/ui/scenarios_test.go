package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// gitOpKeys are the keys of the Git tab that run git in the background (stage, unstage, stash, fetch, pull, push) and
// those that confirm or pick something that does (y, enter). What they change reaches the screen a moment later.
var gitOpKeys = map[string]bool{"s": true, "u": true, "z": true, "f": true, "p": true, "P": true, "y": true, "enter": true}

// An assert right after a git operation reads the screen before git finished: it passed on a quiet machine and failed
// two runs in six with scenarios in parallel (audit of 019e3e5, R1). After such a key a scenario must wait
// for the result with a regex.
func TestNoScenarioAssertsRightAfterAGitOperation(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "tuiqa", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no scenarios: %v", err)
	}
	type step struct {
		Keys   []string        `json:"keys"`
		Assert json.RawMessage `json:"assert"`
	}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var sc struct {
			Steps []step `json:"steps"`
		}
		if err := json.Unmarshal(raw, &sc); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for i := 1; i < len(sc.Steps); i++ {
			prev := sc.Steps[i-1].Keys
			if len(prev) == 1 && gitOpKeys[prev[0]] && sc.Steps[i].Assert != nil {
				t.Errorf("%s step %d: an assert right after %q; wait for the result instead", filepath.Base(f), i, prev[0])
			}
		}
	}
}
