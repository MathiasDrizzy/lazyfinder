package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestASearchCanBeSavedAndRunAgainFromThePicker(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	typeText(e, "kind:text")
	e.send(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	e.mustHave("Search saved")
	e.key("esc") // leave the search
	e.mustHave("data.xlsx")
	e.key("F")
	e.mustHave("Saved searches")
	e.mustHave("kind:text")
	e.key("enter")
	e.mustHave("notes.txt")
	e.mustNotHave("data.xlsx")
}

func TestSavingTheSameSearchTwiceKeepsOne(t *testing.T) {
	e := newEnv(t)
	for i := 0; i < 2; i++ {
		e.key("/")
		typeText(e, "notes")
		e.send(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
		e.key("esc")
	}
	if len(e.m.saved) != 1 {
		t.Fatalf("saved = %+v", e.m.saved)
	}
}

func TestAnEmptySearchIsNotSaved(t *testing.T) {
	e := newEnv(t)
	e.key("/")
	e.send(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if len(e.m.saved) != 0 {
		t.Fatalf("saved = %+v", e.m.saved)
	}
}

func TestDeleteRemovesASavedSearchFromThePicker(t *testing.T) {
	e := newEnv(t)
	e.m.saved = []savedSearch{{Query: "notes"}, {Query: "kind:image"}}
	e.key("F")
	e.key("delete")
	if len(e.m.saved) != 1 || e.m.saved[0].Query != "kind:image" {
		t.Fatalf("saved = %+v", e.m.saved)
	}
	e.mustNotHave("notes ")
	e.key("delete")
	if len(e.m.saved) != 0 || e.m.picker.open {
		t.Fatalf("saved = %+v open = %v", e.m.saved, e.m.picker.open)
	}
}

func TestSavedSearchesSurviveARestart(t *testing.T) {
	e := newEnv(t)
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	d := e.m.deps
	d.ConfigPath = cfgPath
	e.m = New(d)
	e.send(tea.WindowSizeMsg{Width: 120, Height: 40})
	e.key("/")
	typeText(e, "notes")
	e.send(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	data, err := os.ReadFile(filepath.Join(filepath.Dir(cfgPath), "searches.json"))
	if err != nil || !strings.Contains(string(data), "notes") {
		t.Fatalf("searches.json: %v %s", err, data)
	}
	if again := New(d); len(again.saved) != 1 || again.saved[0].Query != "notes" {
		t.Fatalf("after restart: %+v", again.saved)
	}
}
