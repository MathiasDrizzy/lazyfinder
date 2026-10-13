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
	cases := []struct{ name, body, want string }{
		{"malformed", `{"version": 1,`, "line"},
		{"unknown key", `{"version":1,"colums":["name"]}`, "colums"},
		{"bad version", `{"version": 2}`, "version"},
		{"no version", `{"columns":["name"]}`, "version"},
		{"bad column", `{"version":1,"columns":["name","weight"]}`, "weight"},
		{"dangling opener", `{"version":1,"open_with":{"txt":"nope"}}`, "nope"},
		{"file not whole arg", `{"version":1,"openers":{"x":{"argv":["x","--f={file}"],"mode":"gui"}},"open_with":{}}`, "{file}"},
		{"no file placeholder", `{"version":1,"openers":{"x":{"argv":["x"],"mode":"gui"}}}`, "{file}"},
		{"bad mode", `{"version":1,"openers":{"x":{"argv":["x","{file}"],"mode":"weird"}}}`, "mode"},
		{"empty argv", `{"version":1,"openers":{"x":{"argv":[],"mode":"gui"}}}`, "argv"},
		{"bad theme", `{"version":1,"theme":"neon"}`, "theme"},
	}
	for _, c := range cases {
		p := write(t, t.TempDir(), c.body)
		_, err := Load(p, "/home/u")
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want mention of %q", c.name, err, c.want)
		}
	}
}

func TestDefaultJSONRoundTrips(t *testing.T) {
	p := write(t, t.TempDir(), string(DefaultJSON("/home/u")))
	if _, err := Load(p, "/home/u"); err != nil {
		t.Fatalf("default config must be valid: %v", err)
	}
}

func TestOpsDefaultsAndOverrides(t *testing.T) {
	def := Defaults("/home/u")
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
		if _, err := Load(write(t, t.TempDir(), body), "/home/u"); err == nil || !strings.Contains(err.Error(), "ops.") {
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
		if _, err := Load(write(t, t.TempDir(), body), "/home/u"); err == nil || !strings.Contains(err.Error(), "preview.") {
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
		if _, err := Load(write(t, t.TempDir(), body), "/home/u"); err == nil || !strings.Contains(err.Error(), "command") {
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
	if _, err := Load(write(t, t.TempDir(), `{"version":1,"search":{"content_tool":"grep"}}`), "/home/u"); err == nil || !strings.Contains(err.Error(), "search.content_tool") {
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
	if _, err := Load(write(t, t.TempDir(), `{"version":1,"layout":"grid"}`), "/home/u"); err == nil || !strings.Contains(err.Error(), "layout") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidationErrorsSpeakTheInterfaceLanguage(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	want := map[string]string{"es": "debe ser list o columns", "en": "must be list or columns", "pt": "deve ser list ou columns", "fr": "doit être list ou columns",
		"de": "muss list oder columns sein", "it": "deve essere list o columns", "ja": "list か columns です", "zh": "必须是 list 或 columns"}
	for lang, text := range want {
		i18n.SetLanguage(lang)
		_, err := Load(write(t, t.TempDir(), `{"version":1,"layout":"grid"}`), "/home/u")
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
