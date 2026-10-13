package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultsOfTheLook(t *testing.T) {
	cfg := Defaults("/home/u")
	if cfg.Theme != "catppuccin-mocha" || cfg.Language != "auto" || cfg.ScreenBackground != "theme" || !cfg.Mouse {
		t.Fatalf("look defaults wrong: %+v", cfg)
	}
	if len(cfg.Keys) != 0 {
		t.Fatalf("keys must start empty: %v", cfg.Keys)
	}
}

func TestParseLookKeys(t *testing.T) {
	cfg, err := Parse([]byte(`{"version":1,"theme":"nord","language":"es","screen_background":"terminal","mouse":false,
		"keys":{"quit":["x"]}}`), "/h")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != "nord" || cfg.Language != "es" || cfg.ScreenBackground != "terminal" || cfg.Mouse {
		t.Fatalf("parsed wrong: %+v", cfg)
	}
	if got := cfg.Keys["quit"]; len(got) != 1 || got[0] != "x" {
		t.Fatalf("keys = %v", cfg.Keys)
	}
}

func TestThemeTerminalIsAccepted(t *testing.T) {
	if _, err := Parse([]byte(`{"version":1,"theme":"terminal"}`), "/h"); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidLookValues(t *testing.T) {
	cases := map[string]string{
		`{"version":1,"theme":"nope"}`:               `"theme"`,
		`{"version":1,"theme":"dark"}`:               `"theme"`,
		`{"version":1,"language":"klingon"}`:         `"language"`,
		`{"version":1,"screen_background":"purple"}`: `"screen_background"`,
		`{"version":1,"mouse":"yes"}`:                `mouse`,
	}
	for body, want := range cases {
		_, err := Parse([]byte(body), "/h")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want an error mentioning %s, got %v", body, want, err)
		}
	}
}

func TestKeyProblemsAreWarningsNotErrors(t *testing.T) {
	cfg, err := Parse([]byte(`{"version":1,"keys":{"teleport":["t"],"up":["j"]}}`), "/h")
	if err != nil {
		t.Fatalf("bad keys must not stop the app: %v", err)
	}
	if len(cfg.Warnings) != 2 {
		t.Fatalf("want 2 warnings, got %v", cfg.Warnings)
	}
}

func TestLoadReadsUserThemesNextToTheConfig(t *testing.T) {
	dir := t.TempDir()
	themes := filepath.Join(dir, "themes")
	if err := os.Mkdir(themes, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"base":"#0a1014","mantle":"#070b0d","surface0":"#162128","surface1":"#222e36","overlay0":"#758a96",
	"text":"#cccfd1","subtext0":"#a9b0b5","peach":"#99c1dc","mauve":"#b389a7","teal":"#5ba1a3","green":"#7b9f7e",
	"red":"#bf878c","blue":"#7f97be","yellow":"#a1966d"}`
	if err := os.WriteFile(filepath.Join(themes, "night.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themes, "broken.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := write(t, dir, `{"version":1,"theme":"night"}`)
	cfg, err := Load(p, "/h")
	if err != nil {
		t.Fatalf("a user theme must be usable: %v", err)
	}
	if cfg.Theme != "night" || len(cfg.Warnings) != 1 || !strings.Contains(cfg.Warnings[0], "broken") {
		t.Fatalf("theme %q warnings %v", cfg.Theme, cfg.Warnings)
	}
}

func TestUpdateCreatesTheFileWithTheVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	if err := Update(p, map[string]any{"theme": "nord"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p, "/h")
	if err != nil || cfg.Theme != "nord" || cfg.Version != 1 {
		t.Fatalf("round trip failed: %+v %v", cfg, err)
	}
}

func TestUpdateKeepsEverythingElseAndMergesNestedKeys(t *testing.T) {
	dir := t.TempDir()
	p := write(t, dir, `{"version":1,"openers":{"e":{"argv":["code","{file}"],"mode":"gui"}},"open_with":{"txt":"e"},
	"preview":{"max_text_bytes":1000,"images":"auto"},"sidebar":{"favorites":[{"name":"H","path":"~"}],"locations":[]}}`)
	if err := Update(p, map[string]any{"preview.images": "off", "language": "de", "show_hidden": true}); err != nil {
		t.Fatal(err)
	}
	var raw map[string]any
	data, _ := os.ReadFile(p)
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"~"`) {
		t.Errorf("the home shorthand must survive a save: %s", data)
	}
	cfg, err := Load(p, "/h")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Preview.Images != "off" || cfg.Preview.MaxTextBytes != 1000 || cfg.Language != "de" || !cfg.ShowHidden {
		t.Fatalf("not merged: %+v", cfg)
	}
	if cfg.OpenWith["txt"] != "e" || cfg.Openers["e"].Argv[0] != "code" {
		t.Fatalf("openers lost: %+v", cfg)
	}
}

func TestUpdateRefusesToOverwriteAFileItCannotRead(t *testing.T) {
	p := write(t, t.TempDir(), `{ not json`)
	if err := Update(p, map[string]any{"theme": "nord"}); err == nil {
		t.Fatal("must not replace a broken file the user is editing")
	}
	data, _ := os.ReadFile(p)
	if string(data) != `{ not json` {
		t.Fatalf("file changed: %s", data)
	}
}

func TestUpdateLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := Update(p, map[string]any{"mouse": false}); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("only config.json should remain, got %v", entries)
	}
}

func TestPopupBackgroundAndConfirmDeleteDefaultsAndValidation(t *testing.T) {
	d := Defaults("/h")
	if d.PopupBackground != ScreenBackgroundTheme || !d.ConfirmDelete {
		t.Fatalf("defaults: popup_background=%q confirm_delete=%v", d.PopupBackground, d.ConfirmDelete)
	}
	cfg, err := Parse([]byte(`{"version":1,"popup_background":"terminal","confirm_delete":false}`), "/h")
	if err != nil || cfg.PopupBackground != ScreenBackgroundTerminal || cfg.ConfirmDelete {
		t.Fatalf("parse: %+v, %v", cfg, err)
	}
	if _, err := Parse([]byte(`{"version":1,"popup_background":"blue"}`), "/h"); err == nil || !strings.Contains(err.Error(), "popup_background") {
		t.Fatalf("a wrong popup_background must be named: %v", err)
	}
}
