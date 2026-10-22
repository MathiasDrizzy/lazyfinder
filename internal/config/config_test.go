package config

import (
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, body string) string {
	t.Helper()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefaultsWhenFileMissing(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.json"), "/home/u")
	if err != nil {
		t.Fatalf("missing file must mean defaults: %v", err)
	}
	if len(cfg.Columns) != 5 || cfg.Columns[0] != "name" || cfg.Columns[4] != "kind" {
		t.Fatalf("columns = %v", cfg.Columns)
	}
	if !cfg.Sort.FoldersFirst || cfg.ShowHidden {
		t.Fatalf("defaults wrong: %+v", cfg)
	}
	if len(cfg.Sidebar.Favorites) == 0 || cfg.Sidebar.Favorites[0].Path != "/home/u" {
		t.Fatalf("favorites = %+v", cfg.Sidebar.Favorites)
	}
	if cfg.Preview.MaxTextBytes != 524288 {
		t.Fatalf("max text = %d", cfg.Preview.MaxTextBytes)
	}
}

func TestResolvePathOrder(t *testing.T) {
	env := map[string]string{"LAZYFINDER_CONFIG": "/x/c.json", "XDG_CONFIG_HOME": "/xdg"}
	get := func(k string) string { return env[k] }
	if got := ResolvePath(get, "/home/u"); got != "/x/c.json" {
		t.Fatalf("got %q", got)
	}
	delete(env, "LAZYFINDER_CONFIG")
	if got := ResolvePath(get, "/home/u"); got != "/xdg/lazyfinder/config.json" {
		t.Fatalf("got %q", got)
	}
	delete(env, "XDG_CONFIG_HOME")
	if got := ResolvePath(get, "/home/u"); got != "/home/u/.config/lazyfinder/config.json" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadValid(t *testing.T) {
	p := write(t, t.TempDir(), `{
	  "version": 1,
	  "columns": ["name","size"],
	  "openers": {"code": {"argv": ["code","{file}"], "mode": "gui"}},
	  "open_with": {"TXT": "code"},
	  "sidebar": {"favorites": [{"name":"Home","path":"~"}], "locations": []}
	}`)
	cfg, err := Load(p, "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Columns) != 2 {
		t.Fatalf("columns = %v", cfg.Columns)
	}
	if cfg.OpenWith["txt"] != "code" {
		t.Fatalf("extension keys must be lowercased: %v", cfg.OpenWith)
	}
	if cfg.Sidebar.Favorites[0].Path != "/home/u" {
		t.Fatalf("~ not expanded: %+v", cfg.Sidebar.Favorites)
	}
}

func TestInvalidConfigs(t *testing.T) {
	// Only a file that cannot be read as a whole fails; a wrong key is a warning (TestOneWrongKeyDoesNotSpoilTheRest).
	cases := []struct{ name, body, want string }{
		{"malformed", `{"version": 1,`, "line"},
		{"bad version", `{"version": 2}`, "version"},
		{"no version", `{"columns":["name"]}`, "version"},
	}
	for _, c := range cases {
		p := write(t, t.TempDir(), c.body)
		_, err := Load(p, "/home/u")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want mention of %q", c.name, err, c.want)
		}
	}
}

// A wrong value goes back to its default, a warning says which key, the rest of the file is kept.
func TestOneWrongKeyDoesNotSpoilTheRest(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"unknown key", `{"version":1,"colums":["name"]}`, "colums"},
		{"bad column", `{"version":1,"columns":["name","weight"]}`, "weight"},
		{"dangling opener", `{"version":1,"open_with":{"txt":"nope"}}`, "nope"},
		{"file not whole arg", `{"version":1,"openers":{"x":{"argv":["x","--f={file}"],"mode":"gui"}},"open_with":{}}`, "{file}"},
		{"no file placeholder", `{"version":1,"openers":{"x":{"argv":["x"],"mode":"gui"}}}`, "{file}"},
		{"bad mode", `{"version":1,"openers":{"x":{"argv":["x","{file}"],"mode":"weird"}}}`, "mode"},
		{"empty argv", `{"version":1,"openers":{"x":{"argv":[],"mode":"gui"}}}`, "argv"},
		{"bad theme", `{"version":1,"theme":"neon"}`, "theme"},
		{"wrong type", `{"version":1,"mouse":"yes"}`, "mouse"},
	}
	for _, c := range cases {
		body := strings.Replace(c.body, `"version":1,`, `"version":1,"density":"comfortable","sidebar_width":30,`, 1)
		p := write(t, t.TempDir(), body)
		cfg, err := Load(p, "/home/u")
		if err != nil {
			t.Errorf("%s: a wrong key must not fail the file: %v", c.name, err)
			continue
		}
		if cfg.Density != "comfortable" || cfg.SidebarWidth != 30 {
			t.Errorf("%s: the good keys were lost: density=%q sidebar=%d", c.name, cfg.Density, cfg.SidebarWidth)
		}
		if len(cfg.Warnings) != 1 || !strings.Contains(cfg.Warnings[0], c.want) {
			t.Errorf("%s: warnings = %v, want one mentioning %q", c.name, cfg.Warnings, c.want)
		}
	}
	p := write(t, t.TempDir(), `{"version":1,"theme":"neon"}`)
	cfg, _ := Load(p, "/home/u")
	if cfg.Theme != Defaults("/home/u").Theme {
		t.Errorf("a wrong theme goes back to the default, got %q", cfg.Theme)
	}
}

func TestDefaultJSONRoundTrips(t *testing.T) {
	p := write(t, t.TempDir(), string(DefaultJSON("/home/u")))
	if _, err := Load(p, "/home/u"); err != nil {
		t.Fatalf("default config must be valid: %v", err)
	}
}

func TestOpsDefaultsAndOverrides(t *testing.T) {
	def := DefaultsFor("darwin", "/home/u") // platform_test.go checks the Linux trash
	if def.Ops.ConfirmTrashAbove != 1 || def.Ops.TrashDir != "/home/u/.Trash" {
		t.Fatalf("defaults = %+v", def.Ops)
	}
	cfg, err := Load(write(t, t.TempDir(), `{"version":1,"ops":{"confirm_trash_above":0,"trash_dir":"~/bin-can"}}`), "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ops.ConfirmTrashAbove != 0 || cfg.Ops.TrashDir != "/home/u/bin-can" {
		t.Fatalf("ops = %+v", cfg.Ops)
	}
}

func TestOpsRejectsNonsense(t *testing.T) {
	for name, body := range map[string]string{
		"negative threshold": `{"version":1,"ops":{"confirm_trash_above":-1}}`,
		"relative trash":     `{"version":1,"ops":{"trash_dir":"trash"}}`,
		"empty trash":        `{"version":1,"ops":{"trash_dir":""}}`,
	} {
		if _, err := Parse([]byte(body), "/home/u"); err == nil || !strings.Contains(err.Error(), "ops.") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestPreviewExtrasDefaultsAndOverrides(t *testing.T) {
	def := Defaults("/home/u").Preview
	if def.Markdown != "render" || def.HexBytes != 4096 || def.ExternalTools != "auto" {
		t.Fatalf("defaults = %+v", def)
	}
	cfg, err := Load(write(t, t.TempDir(), `{"version":1,"preview":{"markdown":"raw","hex_bytes":256,"external_tools":"off"}}`), "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if p := cfg.Preview; p.Markdown != "raw" || p.HexBytes != 256 || p.ExternalTools != "off" {
		t.Fatalf("preview = %+v", p)
	}
}

func TestPreviewExtrasRejectNonsense(t *testing.T) {
	for name, body := range map[string]string{
		"markdown":       `{"version":1,"preview":{"markdown":"fancy"}}`,
		"hex bytes":      `{"version":1,"preview":{"hex_bytes":0}}`,
		"external tools": `{"version":1,"preview":{"external_tools":"maybe"}}`,
	} {
		if _, err := Parse([]byte(body), "/home/u"); err == nil || !strings.Contains(err.Error(), "preview.") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestCommandsAreParsedAndValidated(t *testing.T) {
	cfg, err := Load(write(t, t.TempDir(), `{"version":1,"commands":{"gs":{"argv":["git","status","{dir}"],"mode":"terminal"},"wc":{"argv":["wc","-l","{file}"],"mode":"terminal"},"make":{"argv":["make"],"mode":"gui"}}}`), "/home/u")
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.CommandNames(); strings.Join(got, ",") != "gs,make,wc" {
		t.Fatalf("names = %v", got)
	}
	for name, body := range map[string]string{
		"no argv":          `{"version":1,"commands":{"x":{"argv":[],"mode":"gui"}}}`,
		"bad mode":         `{"version":1,"commands":{"x":{"argv":["ls"],"mode":"fancy"}}}`,
		"placeholder part": `{"version":1,"commands":{"x":{"argv":["ls","a{file}"],"mode":"gui"}}}`,
		"program name":     `{"version":1,"commands":{"x":{"argv":["{file}"],"mode":"gui"}}}`,
		"twice":            `{"version":1,"commands":{"x":{"argv":["ls","{file}","{file}"],"mode":"gui"}}}`,
	} {
		if _, err := Parse([]byte(body), "/home/u"); err == nil || !strings.Contains(err.Error(), "command") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestSearchContentToolDefaultsAndValidation(t *testing.T) {
	if got := Defaults("/home/u").Search.ContentTool; got != "auto" {
		t.Fatalf("default = %q", got)
	}
	cfg, err := Load(write(t, t.TempDir(), `{"version":1,"search":{"content_tool":"native"}}`), "/home/u")
	if err != nil || cfg.Search.ContentTool != "native" {
		t.Fatalf("cfg = %+v err = %v", cfg.Search, err)
	}
	if _, err := Parse([]byte(`{"version":1,"search":{"content_tool":"grep"}}`), "/home/u"); err == nil || !strings.Contains(err.Error(), "search.content_tool") {
		t.Fatalf("err = %v", err)
	}
}

func TestLayoutDefaultAndValidation(t *testing.T) {
	if got := Defaults("/home/u").Layout; got != "list" {
		t.Fatalf("default = %q", got)
	}
	if cfg, err := Load(write(t, t.TempDir(), `{"version":1,"layout":"columns"}`), "/home/u"); err != nil || cfg.Layout != "columns" {
		t.Fatalf("cfg = %q err = %v", cfg.Layout, err)
	}
	if _, err := Parse([]byte(`{"version":1,"layout":"grid"}`), "/home/u"); err == nil || !strings.Contains(err.Error(), "layout") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidationErrorsSpeakTheInterfaceLanguage(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	want := map[string]string{"es": "debe ser list o columns", "en": "must be list or columns", "pt": "deve ser list ou columns", "fr": "doit être list ou columns",
		"de": "muss list oder columns sein", "it": "deve essere list o columns", "ja": "list か columns です", "zh": "必须是 list 或 columns"}
	for lang, text := range want {
		i18n.SetLanguage(lang)
		_, err := Parse([]byte(`{"version":1,"layout":"grid"}`), "/home/u")
		if err == nil || !strings.Contains(err.Error(), text) {
			t.Errorf("%s: err = %v, want it to contain %q", lang, err, text)
		}
	}
}

func TestSearchScopeKindsAndByDefaultAndValidation(t *testing.T) {
	s := Defaults("/home/u").Search
	if s.Scope != "folder" || s.Kinds != "both" || s.By != "name" {
		t.Fatalf("defaults = %+v", s)
	}
	for _, js := range []string{`{"version":1,"search":{"scope":"system","kinds":"files","by":"content"}}`} {
		cfg, err := Parse([]byte(js), "/home/u")
		if err != nil || cfg.Search.Scope != "system" || cfg.Search.Kinds != "files" || cfg.Search.By != "content" {
			t.Fatalf("%v %+v", err, cfg.Search)
		}
	}
	for _, key := range []string{"scope", "kinds", "by"} {
		_, err := Parse([]byte(`{"version":1,"search":{"`+key+`":"nope"}}`), "/home/u")
		if err == nil || !strings.Contains(err.Error(), "search."+key) {
			t.Errorf("%s: err = %v", key, err)
		}
	}
}

func TestLayoutSizesDefaultAndValidation(t *testing.T) {
	d := Defaults("/home/u")
	if d.SidebarWidth != 0 || d.PreviewWidth != 0 || len(d.ColumnWidths) != 0 {
		t.Fatalf("0 means automatic: %+v", d)
	}
	cfg, err := Parse([]byte(`{"version":1,"sidebar_width":30,"preview_width":50,"column_widths":{"size":12,"kind":24}}`), "/home/u")
	if err != nil || cfg.SidebarWidth != 30 || cfg.PreviewWidth != 50 || cfg.ColumnWidths["kind"] != 24 {
		t.Fatalf("%v %+v", err, cfg)
	}
	for _, js := range []string{
		`{"version":1,"sidebar_width":5}`, `{"version":1,"sidebar_width":99}`, `{"version":1,"preview_width":3}`,
		`{"version":1,"column_widths":{"name":20}}`, `{"version":1,"column_widths":{"size":1}}`, `{"version":1,"column_widths":{"nope":10}}`,
	} {
		if _, err := Parse([]byte(js), "/home/u"); err == nil {
			t.Errorf("%s should be refused", js)
		}
	}
}

func TestKeybindingModeValidation(t *testing.T) {
	if d := Defaults("/home/u"); d.KeybindingMode != "dual" {
		t.Fatalf("default %q", d.KeybindingMode)
	}
	cfg, err := Parse([]byte(`{"version":1,"keybinding_mode":"lazy"}`), "/home/u")
	if err != nil || cfg.KeybindingMode != "lazy" {
		t.Fatalf("%v %q", err, cfg.KeybindingMode)
	}
	if _, err := Parse([]byte(`{"version":1,"keybinding_mode":"emacs"}`), "/home/u"); err == nil {
		t.Fatal("emacs should be refused")
	}
}

func TestMoreColumnsAndDateFormat(t *testing.T) {
	cfg, err := Parse([]byte(`{"version":1,"columns":["name","opened","extension","permissions","tags","size"],"date_format":"numeric","density":"comfortable"}`), "/home/u")
	if err != nil || cfg.DateFormat != "numeric" || cfg.Density != "comfortable" {
		t.Fatalf("%v %+v", err, cfg)
	}
	d := Defaults("/home/u")
	if d.DateFormat != "absolute" || d.Density != "comfortable" {
		t.Fatalf("defaults %q %q", d.DateFormat, d.Density)
	}
	for _, js := range []string{`{"version":1,"date_format":"iso"}`, `{"version":1,"density":"huge"}`, `{"version":1,"columns":["name","location"]}`} {
		if _, err := Parse([]byte(js), "/home/u"); err == nil {
			t.Errorf("%s should be refused", js)
		}
	}
}

// Order 2026-10-15-16: inside herdr the sidebar opens by itself at the edge chosen in the settings. Mathias,
// 2026-10-22: every tab opens it at the same width in columns, one for the computer's screen and one for an external
// monitor ("los dos deben ser configurable en configuraciones"), never narrower than a minimum.
func TestHerdrDefaultsAndValidation(t *testing.T) {
	h := Defaults("/home/u").Herdr
	want := Herdr{Dock: "left", AutoOpen: true, Columns: 30, ExternalColumns: 30, ExternalFrom: 200, Adapt: true, MinColumns: HerdrMinColumns, FitNormal: true, OpenIn: "tab"}
	if h != want {
		t.Fatalf("defaults = %+v", h)
	}
	cfg, err := Parse([]byte(`{"version":1,"herdr":{"dock":"right","auto_open":false,"columns":26,"external_columns":40,"external_from":180,"adapt":false,"min_columns":24,"fit_normal":false,"preview":true,"open_in":"pane"}}`), "/home/u")
	want = Herdr{Dock: "right", Columns: 26, ExternalColumns: 40, ExternalFrom: 180, MinColumns: 24, Preview: true, OpenIn: "pane"}
	if err != nil || cfg.Herdr != want {
		t.Fatalf("%v %+v", err, cfg.Herdr)
	}
	for _, js := range []string{
		`{"version":1,"herdr":{"dock":"top"}}`,
		`{"version":1,"herdr":{"columns":5}}`,
		`{"version":1,"herdr":{"external_columns":500}}`,
		`{"version":1,"herdr":{"min_columns":10}}`,
		`{"version":1,"herdr":{"external_from":0}}`,
		`{"version":1,"herdr":{"open_in":"window"}}`,
	} {
		if _, err := Parse([]byte(js), "/home/u"); err == nil || !strings.Contains(err.Error(), "herdr.") {
			t.Errorf("%s: err = %v", js, err)
		}
	}
}

// herdr.width (percent of the tab) gave way to herdr.columns: a config.json that still has it loads, and the width
// comes from the columns.
func TestTheOldHerdrWidthStillLoads(t *testing.T) {
	cfg, err := Parse([]byte(`{"version":1,"herdr":{"dock":"left","width":25}}`), "/home/u")
	if err != nil || cfg.Herdr.Columns != 30 {
		t.Fatalf("%v %+v", err, cfg.Herdr)
	}
}
