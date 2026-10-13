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

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
)

// FilePlaceholder is the only placeholder in an opener's argv; it must be a whole element.
const FilePlaceholder = "{file}"

// DirPlaceholder is the current folder in a custom command's argv; like {file} it must be a whole element.
const DirPlaceholder = "{dir}"

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
	Commands   map[string]Opener `json:"commands"`
	OpenWith   map[string]string `json:"open_with"`
	Preview    Preview           `json:"preview"`
	Layout     string            `json:"layout"` // list (detail columns) or columns (parent, folder, preview)
	Theme      string            `json:"theme"`
	Watch      Watch             `json:"watch"`
	Git        Git               `json:"git"`
	Search     Search            `json:"search"`
	Ops        Ops               `json:"ops"`

	// Language is "auto" or a code the i18n package knows. ScreenBackground is "theme" or "terminal".
	Language         string              `json:"language"`
	ScreenBackground string              `json:"screen_background"`
	Mouse            bool                `json:"mouse"`
	Keys             map[string][]string `json:"keys"`

	// Warnings are problems that do not stop the app (a broken user theme, a clashing key). Not part of the file.
	Warnings []string `json:"-"`
}

// Values of ScreenBackground.
const (
	ScreenBackgroundTheme    = "theme"
	ScreenBackgroundTerminal = "terminal"
)

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
	MaxTextBytes  int64  `json:"max_text_bytes"`
	Images        string `json:"images"`
	Markdown      string `json:"markdown"`       // render or raw
	HexBytes      int    `json:"hex_bytes"`      // bytes of a binary file shown as hex
	ExternalTools string `json:"external_tools"` // auto lets previews use mdls, pdftoppm or sips; off never runs them
}

// Search configures the search bar.
type Search struct {
	MaxResults  int    `json:"max_results"`
	ContentTool string `json:"content_tool"` // auto uses ripgrep for in: when installed; native never runs it
}

// Ops configures file operations.
type Ops struct {
	// ConfirmTrashAbove asks before trashing more than this many items at once (0 asks always).
	ConfirmTrashAbove int `json:"confirm_trash_above"`
	// TrashDir is the folder that receives trashed items; the macOS Trash by default.
	TrashDir string `json:"trash_dir"`
}

// Git configures the status markers and the branch name.
type Git struct {
	Enabled     bool `json:"enabled"`
	HideIgnored bool `json:"hide_ignored"`
}

// Watch configures live follow.
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
		Commands: map[string]Opener{},
		OpenWith: map[string]string{},
		Preview:  Preview{MaxTextBytes: defaultMaxTextBytes, Images: "auto", Markdown: "render", HexBytes: 4096, ExternalTools: "auto"},
		Layout:   "list",
		Theme:    theme.Default,
		Watch:    Watch{MaxDirs: 64, PollIntervalMS: 2000},
		Git:      Git{Enabled: true},
		Search:   Search{MaxResults: 5000, ContentTool: "auto"},
		Ops:      Ops{ConfirmTrashAbove: 1, TrashDir: filepath.Join(home, ".Trash")},

		Language:         "auto",
		ScreenBackground: ScreenBackgroundTheme,
		Mouse:            true,
		Keys:             map[string][]string{},
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
		return withThemes(Defaults(home), path), nil
	}
	if err != nil {
		return Config{}, i18n.Errorf(`no se puede leer la configuración %s: %w`, `cannot read config %s: %w`, path, err)
	}
	// The user's themes load before validation: the config may name one of them.
	var warnings []string
	for _, p := range theme.LoadUserThemes(filepath.Join(filepath.Dir(path), "themes")) {
		warnings = append(warnings, fmt.Sprintf(p.EN, p.Args...))
	}
	cfg, err := Parse(data, home)
	if err != nil {
		return Config{}, i18n.Errorf(`configuración no válida: %w (archivo %s)`, `invalid config: %w (file %s)`, err, path)
	}
	cfg.Warnings = append(warnings, cfg.Warnings...)
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
		return Config{}, i18n.Errorf(`"version" debe ser %d (hay %d)`, `"version" must be %d (got %d)`, currentVersion, cfg.Version)
	}
	return cfg, validate(&cfg, home)
}

func describeJSON(err error, data []byte) error {
	var se *json.SyntaxError
	if errors.As(err, &se) {
		line := 1 + bytes.Count(data[:min(int(se.Offset), len(data))], []byte("\n"))
		return i18n.Errorf(`JSON no válido en la línea %d: %v`, `invalid JSON at line %d: %v`, line, err)
	}
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		return i18n.Errorf(`el campo %q tiene un tipo incorrecto (se espera %s)`, `field %q has the wrong type (want %s)`, ute.Field, ute.Type)
	}
	if errors.Is(err, errors.ErrUnsupported) || strings.Contains(err.Error(), "unexpected EOF") {
		return i18n.Errorf(`JSON no válido en la línea %d: %v`, `invalid JSON at line %d: %v`, 1+bytes.Count(data, []byte("\n")), err)
	}
	return err
}

func validate(cfg *Config, home string) error {
	if len(cfg.Columns) == 0 {
		return i18n.NewError(`"columns" no puede estar vacío`, `"columns" must not be empty`)
	}
	for _, c := range cfg.Columns {
		if !validColumns[c] {
			return i18n.Errorf(`columna desconocida %q en "columns"`, `unknown column %q in "columns"`, c)
		}
	}
	if !validColumns[cfg.Sort.By] {
		return i18n.Errorf(`columna desconocida %q en "sort.by"`, `unknown "sort.by" column %q`, cfg.Sort.By)
	}
	if !theme.Valid(cfg.Theme) {
		return i18n.Errorf(`"theme" %q no es un tema (usa %s, o pon %s.json en la carpeta de temas)`, `"theme" %q is not a theme (use %s, or put %s.json in the themes folder)`, cfg.Theme,
			strings.Join(theme.Names(), ", "), cfg.Theme)
	}
	if cfg.Language != "auto" && !validLanguage(cfg.Language) {
		return i18n.Errorf(`"language" debe ser auto, en, es, pt, fr, de, it, ja o zh (hay %q)`, `"language" must be auto, en, es, pt, fr, de, it, ja or zh (got %q)`, cfg.Language)
	}
	switch cfg.ScreenBackground {
	case ScreenBackgroundTheme, ScreenBackgroundTerminal:
	default:
		return i18n.Errorf(`"screen_background" debe ser theme o terminal (hay %q)`, `"screen_background" must be theme or terminal (got %q)`, cfg.ScreenBackground)
	}
	_, cfg.Warnings = keymap.New(cfg.Keys)
	switch cfg.Preview.Images {
	case "auto", "kitty", "off":
	default:
		return i18n.Errorf(`"preview.images" debe ser auto, kitty u off (hay %q)`, `"preview.images" must be auto, kitty or off (got %q)`, cfg.Preview.Images)
	}
	if cfg.Preview.Markdown != "render" && cfg.Preview.Markdown != "raw" {
		return i18n.Errorf(`"preview.markdown" debe ser render o raw (hay %q)`, `"preview.markdown" must be render or raw (got %q)`, cfg.Preview.Markdown)
	}
	if cfg.Preview.HexBytes <= 0 {
		return i18n.NewError(`"preview.hex_bytes" debe ser positivo`, `"preview.hex_bytes" must be positive`)
	}
	if cfg.Preview.ExternalTools != "auto" && cfg.Preview.ExternalTools != "off" {
		return i18n.Errorf(`"preview.external_tools" debe ser auto u off (hay %q)`, `"preview.external_tools" must be auto or off (got %q)`, cfg.Preview.ExternalTools)
	}
	if cfg.Layout != "list" && cfg.Layout != "columns" {
		return i18n.Errorf(`"layout" debe ser list o columns (hay %q)`, `"layout" must be list or columns (got %q)`, cfg.Layout)
	}
	if cfg.Search.ContentTool != "auto" && cfg.Search.ContentTool != "native" {
		return i18n.Errorf(`"search.content_tool" debe ser auto o native (hay %q)`, `"search.content_tool" must be auto or native (got %q)`, cfg.Search.ContentTool)
	}
	if cfg.Search.MaxResults <= 0 {
		return i18n.NewError(`"search.max_results" debe ser positivo`, `"search.max_results" must be positive`)
	}
	if cfg.Ops.ConfirmTrashAbove < 0 {
		return i18n.NewError(`"ops.confirm_trash_above" no puede ser negativo`, `"ops.confirm_trash_above" must not be negative`)
	}
	if !strings.HasPrefix(cfg.Ops.TrashDir, "~") && !strings.HasPrefix(cfg.Ops.TrashDir, "$HOME") && !filepath.IsAbs(cfg.Ops.TrashDir) {
		return i18n.NewError(`"ops.trash_dir" debe ser una ruta absoluta o empezar por ~`, `"ops.trash_dir" must be an absolute path or start with ~`)
	}
	if cfg.Preview.MaxTextBytes <= 0 {
		return i18n.NewError(`"preview.max_text_bytes" debe ser positivo`, `"preview.max_text_bytes" must be positive`)
	}
	for name, op := range cfg.Openers {
		if err := validateOpener(name, op); err != nil {
			return err
		}
	}
	for name, c := range cfg.Commands {
		if err := validateCommand(name, c); err != nil {
			return err
		}
	}
	norm := make(map[string]string, len(cfg.OpenWith))
	for ext, name := range cfg.OpenWith {
		if _, ok := cfg.Opener(name); !ok {
			return i18n.Errorf(`"open_with.%s" usa el programa %q, que no está definido en "openers"`, `"open_with.%s" refers to opener %q, which is not defined in "openers"`, ext, name)
		}
		norm[strings.ToLower(strings.TrimPrefix(ext, "."))] = name
	}
	cfg.OpenWith = norm
	cfg.Ops.TrashDir = expandHome(cfg.Ops.TrashDir, home)
	cfg.Sidebar.Favorites = expandPlaces(cfg.Sidebar.Favorites, home)
	cfg.Sidebar.Locations = expandPlaces(cfg.Sidebar.Locations, home)
	return nil
}

func validateOpener(name string, op Opener) error {
	if len(op.Argv) == 0 || op.Argv[0] == "" {
		return i18n.Errorf(`programa %q: "argv" debe tener al menos el nombre del programa`, `opener %q: "argv" must have at least the program name`, name)
	}
	if op.Mode != ModeGUI && op.Mode != ModeTerminal {
		return i18n.Errorf(`programa %q: "mode" debe ser %q o %q (hay %q)`, `opener %q: "mode" must be %q or %q (got %q)`, name, ModeGUI, ModeTerminal, op.Mode)
	}
	count := 0
	for i, a := range op.Argv {
		if a == FilePlaceholder {
			if i == 0 {
				return i18n.Errorf(`programa %q: %s no puede ser el nombre del programa`, `opener %q: %s cannot be the program name`, name, FilePlaceholder)
			}
			count++
		} else if strings.Contains(a, FilePlaceholder) {
			return i18n.Errorf(`programa %q: %s debe ser un elemento completo de argv, no parte de %q`, `opener %q: %s must be a whole argv element, not part of %q`, name, FilePlaceholder, a)
		}
	}
	if count != 1 {
		return i18n.Errorf(`programa %q: "argv" debe contener %s exactamente una vez`, `opener %q: "argv" must contain %s exactly once`, name, FilePlaceholder)
	}
	return nil
}

// validateCommand checks a custom command: a program, a mode, and {file} and {dir} (each optional, at most once,
// whole elements, never the program).
func validateCommand(name string, c Opener) error {
	if len(c.Argv) == 0 || c.Argv[0] == "" {
		return i18n.Errorf(`comando %q: "argv" debe tener al menos el nombre del programa`, `command %q: "argv" must have at least the program name`, name)
	}
	if c.Mode != ModeGUI && c.Mode != ModeTerminal {
		return i18n.Errorf(`comando %q: "mode" debe ser %q o %q (hay %q)`, `command %q: "mode" must be %q or %q (got %q)`, name, ModeGUI, ModeTerminal, c.Mode)
	}
	for _, ph := range []string{FilePlaceholder, DirPlaceholder} {
		count := 0
		for i, a := range c.Argv {
			switch {
			case a == ph && i == 0:
				return i18n.Errorf(`comando %q: %s no puede ser el nombre del programa`, `command %q: %s cannot be the program name`, name, ph)
			case a == ph:
				count++
			case strings.Contains(a, ph):
				return i18n.Errorf(`comando %q: %s debe ser un elemento completo de argv, no parte de %q`, `command %q: %s must be a whole argv element, not part of %q`, name, ph, a)
			}
		}
		if count > 1 {
			return i18n.Errorf(`comando %q: %s solo puede aparecer una vez`, `command %q: %s may appear only once`, name, ph)
		}
	}
	return nil
}

// CommandNames returns the custom command names sorted.
func (c Config) CommandNames() []string {
	names := make([]string, 0, len(c.Commands))
	for n := range c.Commands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
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

var languageCodes = map[string]bool{"en": true, "es": true, "pt": true, "fr": true, "de": true, "it": true, "ja": true, "zh": true}

func validLanguage(code string) bool { return languageCodes[code] }

// withThemes loads the user themes next to path and records what could not be loaded.
func withThemes(cfg Config, path string) Config {
	for _, p := range theme.LoadUserThemes(filepath.Join(filepath.Dir(path), "themes")) {
		cfg.Warnings = append(cfg.Warnings, fmt.Sprintf(p.EN, p.Args...))
	}
	return cfg
}

// Update writes the given settings into the config file at path and leaves everything else in it as it was
// (openers, sidebar, "~" shorthands). Keys are dotted for nested objects ("preview.images"). The file is replaced
// atomically. A file that cannot be read as JSON is never overwritten, because the user may be editing it.
func Update(path string, set map[string]any) error {
	doc := map[string]any{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &doc); err != nil || doc == nil {
			return i18n.Errorf(`no se guarda: %s no es un JSON válido, corrígelo primero`, `not saving: %s is not valid JSON, fix it first`, path)
		}
	case !errors.Is(err, fs.ErrNotExist):
		return i18n.Errorf(`no se puede leer la configuración %s: %w`, `cannot read config %s: %w`, path, err)
	}
	if _, ok := doc["version"]; !ok {
		doc["version"] = currentVersion
	}
	for key, val := range set {
		parts := strings.Split(key, ".")
		node := doc
		for _, part := range parts[:len(parts)-1] {
			next, ok := node[part].(map[string]any)
			if !ok {
				next = map[string]any{}
				node[part] = next
			}
			node = next
		}
		node[parts[len(parts)-1]] = val
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(out, '\n'))
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return i18n.Errorf(`no se puede crear %s: %w`, `cannot create %s: %w`, dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return i18n.Errorf(`no se puede escribir la configuración: %w`, `cannot write config: %w`, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // a no-op after the rename
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return i18n.Errorf(`no se puede escribir la configuración: %w`, `cannot write config: %w`, err)
	}
	if err := tmp.Close(); err != nil {
		return i18n.Errorf(`no se puede escribir la configuración: %w`, `cannot write config: %w`, err)
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
