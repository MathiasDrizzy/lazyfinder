// Package config loads and validates config.json (docs/adr/0004-configuration.md).
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FilePlaceholder is the only placeholder in an opener's argv; it must be a whole element.
const FilePlaceholder = "{file}"

// Opener modes.
const (
	ModeGUI      = "gui"
	ModeTerminal = "terminal"
)

const (
	currentVersion      = 1
	defaultMaxTextBytes = 512 << 10
)

// Config is the validated configuration. Treat it as read-only.
type Config struct {
	Version    int               `json:"version"`
	Columns    []string          `json:"columns"`
	Sort       SortConfig        `json:"sort"`
	ShowHidden bool              `json:"show_hidden"`
	Sidebar    Sidebar           `json:"sidebar"`
	Openers    map[string]Opener `json:"openers"`
	OpenWith   map[string]string `json:"open_with"`
	Preview    Preview           `json:"preview"`
	Theme      string            `json:"theme"`
	Watch      Watch             `json:"watch"`
}

// SortConfig is the initial ordering.
type SortConfig struct {
	By           string `json:"by"`
	Descending   bool   `json:"descending"`
	FoldersFirst bool   `json:"folders_first"`
}

// Sidebar holds the sidebar sections.
type Sidebar struct {
	Favorites []Place `json:"favorites"`
	Locations []Place `json:"locations"`
}

// Place is a sidebar entry.
type Place struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Opener is a program the user configured to open files.
type Opener struct {
	Argv []string `json:"argv"`
	Mode string   `json:"mode"`
}

// Preview configures the preview popup.
type Preview struct {
	MaxTextBytes int64  `json:"max_text_bytes"`
	Images       string `json:"images"`
}

// Watch configures live follow (used from milestone 2).
type Watch struct {
	MaxDirs        int `json:"max_dirs"`
	PollIntervalMS int `json:"poll_interval_ms"`
}

var validColumns = map[string]bool{"name": true, "modified": true, "created": true, "size": true, "kind": true}

// ResolvePath picks the config file: $LAZYFINDER_CONFIG, $XDG_CONFIG_HOME/lazyfinder/config.json,
// then ~/.config/lazyfinder/config.json.
func ResolvePath(getenv func(string) string, home string) string {
	if p := getenv("LAZYFINDER_CONFIG"); p != "" {
		return p
	}
	if x := getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "lazyfinder", "config.json")
	}
	return filepath.Join(home, ".config", "lazyfinder", "config.json")
}

// Defaults returns the configuration used when there is no file.
func Defaults(home string) Config {
	return Config{
		Version:  currentVersion,
		Columns:  []string{"name", "modified", "created", "size", "kind"},
		Sort:     SortConfig{By: "name", FoldersFirst: true},
		Sidebar:  defaultSidebar(home),
		Openers:  map[string]Opener{},
		OpenWith: map[string]string{},
		Preview:  Preview{MaxTextBytes: defaultMaxTextBytes, Images: "auto"},
		Theme:    "auto",
		Watch:    Watch{MaxDirs: 64, PollIntervalMS: 2000},
	}
}

func defaultSidebar(home string) Sidebar {
	j := func(n string) string { return filepath.Join(home, n) }
	return Sidebar{
		Favorites: []Place{
			{"Home", home}, {"Desktop", j("Desktop")}, {"Documents", j("Documents")},
			{"Downloads", j("Downloads")}, {"Applications", "/Applications"},
		},
		Locations: []Place{{"Macintosh HD", "/"}, {"Volumes", "/Volumes"}},
	}
}

// DefaultJSON is the default configuration as JSON, for --print-default-config.
func DefaultJSON(home string) []byte {
	c := Defaults(home)
	b, _ := json.MarshalIndent(c, "", "  ")
	return append(b, '\n')
}

// Load reads path. A missing file yields the defaults; any other problem is an error that names the field or line.
func Load(path, home string) (Config, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Defaults(home), nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("cannot read config %s: %w", path, err)
	}
	cfg, err := Parse(data, home)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config: %w (file %s)", err, path)
	}
	return cfg, nil
}

// Parse validates data and fills in defaults for omitted fields.
func Parse(data []byte, home string) (Config, error) {
	cfg := Defaults(home)
	cfg.Version = 0
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, describeJSON(err, data)
	}
	if cfg.Version != currentVersion {
		return Config{}, fmt.Errorf(`"version" must be %d (got %d)`, currentVersion, cfg.Version)
	}
	return cfg, validate(&cfg, home)
}

func describeJSON(err error, data []byte) error {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		line := 1 + bytes.Count(data[:min(int(se.Offset), len(data))], []byte("\n"))
		return fmt.Errorf("invalid JSON at line %d: %v", line, err)
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		return fmt.Errorf("field %q has the wrong type (want %s)", ute.Field, ute.Type)
	}
	if errors.Is(err, errors.ErrUnsupported) || strings.Contains(err.Error(), "unexpected EOF") {
		return fmt.Errorf("invalid JSON at line %d: %v", 1+bytes.Count(data, []byte("\n")), err)
	}
	return err
}

func validate(cfg *Config, home string) error {
	if len(cfg.Columns) == 0 {
		return errors.New(`"columns" must not be empty`)
	}
	for _, c := range cfg.Columns {
		if !validColumns[c] {
			return fmt.Errorf(`unknown column %q in "columns"`, c)
		}
	}
	if !validColumns[cfg.Sort.By] {
		return fmt.Errorf(`unknown "sort.by" column %q`, cfg.Sort.By)
	}
	switch cfg.Theme {
	case "auto", "light", "dark":
	default:
		return fmt.Errorf(`"theme" must be auto, light or dark (got %q)`, cfg.Theme)
	}
	switch cfg.Preview.Images {
	case "auto", "kitty", "off":
	default:
		return fmt.Errorf(`"preview.images" must be auto, kitty or off (got %q)`, cfg.Preview.Images)
	}
	if cfg.Preview.MaxTextBytes <= 0 {
		return errors.New(`"preview.max_text_bytes" must be positive`)
	}
	for name, op := range cfg.Openers {
		if err := validateOpener(name, op); err != nil {
			return err
		}
	}
	norm := make(map[string]string, len(cfg.OpenWith))
	for ext, name := range cfg.OpenWith {
		if _, ok := cfg.Openers[name]; !ok {
			return fmt.Errorf(`"open_with.%s" refers to opener %q, which is not defined in "openers"`, ext, name)
		}
		norm[strings.ToLower(strings.TrimPrefix(ext, "."))] = name
	}
	cfg.OpenWith = norm
	cfg.Sidebar.Favorites = expandPlaces(cfg.Sidebar.Favorites, home)
	cfg.Sidebar.Locations = expandPlaces(cfg.Sidebar.Locations, home)
	return nil
}

func validateOpener(name string, op Opener) error {
	if len(op.Argv) == 0 || op.Argv[0] == "" {
		return fmt.Errorf(`opener %q: "argv" must have at least the program name`, name)
	}
	if op.Mode != ModeGUI && op.Mode != ModeTerminal {
		return fmt.Errorf(`opener %q: "mode" must be %q or %q (got %q)`, name, ModeGUI, ModeTerminal, op.Mode)
	}
	count := 0
	for i, a := range op.Argv {
		if a == FilePlaceholder {
			if i == 0 {
				return fmt.Errorf(`opener %q: %s cannot be the program name`, name, FilePlaceholder)
			}
			count++
		} else if strings.Contains(a, FilePlaceholder) {
			return fmt.Errorf(`opener %q: %s must be a whole argv element, not part of %q`, name, FilePlaceholder, a)
		}
	}
	if count != 1 {
		return fmt.Errorf(`opener %q: "argv" must contain %s exactly once`, name, FilePlaceholder)
	}
	return nil
}

func expandPlaces(ps []Place, home string) []Place {
	out := make([]Place, len(ps))
	for i, p := range ps {
		out[i] = Place{Name: p.Name, Path: expandHome(p.Path, home)}
	}
	return out
}

func expandHome(p, home string) string {
	switch {
	case p == "~" || p == "$HOME":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, p[2:])
	case strings.HasPrefix(p, "$HOME/"):
		return filepath.Join(home, p[6:])
	}
	return p
}

// OpenerNames returns the configured opener names sorted.
func (c Config) OpenerNames() []string {
	names := make([]string, 0, len(c.Openers))
	for n := range c.Openers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
