// Package icons picks the glyph and the color role of a file or folder: Nerd Font glyphs (the default), plain ASCII
// or nothing. The mapping is data, so a user can override an extension from config.json.
package icons

import (
	"path/filepath"
	"strings"
)

// Style says which glyph family is drawn.
type Style string

const (
	Nerd  Style = "nerd"  // Nerd Font glyphs, one cell each, colored by kind (the herdr-sidebar table, see NOTICE)
	Emoji Style = "emoji" // one emoji per kind, for terminals without a Nerd Font
	ASCII Style = "ascii" // a plain character per kind, for terminals without a Nerd Font
	Off   Style = "off"
)

// Role is a color of the theme palette; the UI maps it to the real color so every theme keeps its look.
type Role int

const (
	Plain Role = iota
	Folder
	Green
	Peach
	Red
	Yellow
	Mauve
	Teal
	Blue
)

// Icon is a glyph with the palette role that colors it; Dim marks hidden items.
type Icon struct {
	Glyph string
	Role  Role
	Dim   bool
	// RGB (0xRRGGBB) is the color of the kind when it has its own, instead of a palette Role; the UI lifts it where
	// a theme would make it unreadable.
	RGB     uint32
	HasRGB  bool
	IsEmoji bool // drawn with its own colors, two cells wide
}

// Item is what the icon is picked for.
type Item struct {
	Name   string
	IsDir  bool
	IsLink bool
	Open   bool // an expanded folder
}

// Set is the configured icon family and the user's extension overrides.
type Set struct {
	Style Style
	Map   map[string]string // extension (with or without the dot, any case) -> glyph
}

// Width is the cells the icon takes with its trailing space: 0 when off.
func (s Set) Width() int {
	switch s.Style {
	case Off, "":
		return 0
	case Emoji:
		return 3 // an emoji is two cells wide
	}
	return 2
}

// Pick returns the icon of an item.
func (s Set) Pick(it Item) Icon {
	if s.Style == Off || s.Style == "" {
		return Icon{}
	}
	hidden := strings.HasPrefix(it.Name, ".")
	if s.Style == ASCII {
		return Icon{Glyph: asciiGlyph(it), Role: roleFor(it), Dim: hidden}
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(it.Name), "."))
	if !it.IsDir {
		if g, ok := s.override(it.Name, ext); ok {
			return Icon{Glyph: g, Role: roleFor(it), Dim: hidden}
		}
	}
	if s.Style == Emoji {
		return Icon{Glyph: emojiOf(it), Role: roleFor(it), Dim: hidden, IsEmoji: true}
	}
	ic := nerd(it, ext)
	ic.Dim = hidden
	if k, ok := kindOf(it.Name); ok && k == kGit && !it.IsDir {
		ic.Dim = false // .gitignore and friends keep their red, like in the herdr sidebar
	}
	return ic
}

// override finds the user's glyph for a file: by its whole name (Makefile, .gitignore) or by its extension.
func (s Set) override(name, ext string) (string, bool) {
	lowerName := strings.ToLower(name)
	for k, g := range s.Map {
		key := strings.ToLower(k)
		if key == lowerName || (ext != "" && strings.TrimPrefix(key, ".") == ext) {
			return g, true
		}
	}
	return "", false
}

func emojiOf(it Item) string {
	switch {
	case it.IsLink:
		return "🔗"
	case it.IsDir && it.Open:
		return kindEmoji[kDirOpen]
	case it.IsDir:
		return kindEmoji[kDir]
	}
	if k, ok := kindOf(it.Name); ok {
		return kindEmoji[k]
	}
	return kindEmoji[kFile]
}

func asciiGlyph(it Item) string {
	switch {
	case it.IsLink:
		return "@"
	case it.IsDir:
		return "/"
	}
	return "-"
}

func roleFor(it Item) Role {
	switch {
	case it.IsDir:
		return Folder
	case it.IsLink:
		return Teal
	}
	return Plain
}

const (
	glyphFolder     = ""
	glyphFolderOpen = ""
	glyphFile       = ""
	glyphText       = ""
	glyphLink       = ""
	glyphApp        = ""
)

type kind struct {
	glyph string
	role  Role
}

// byExt is the table of the Nerd Font kinds (Font Awesome and devicons code points, as lazymark draws its tree).
var byExt = map[string]kind{
	"png": {"", Mauve}, "jpg": {"", Mauve}, "jpeg": {"", Mauve}, "gif": {"", Mauve}, "webp": {"", Mauve},
	"heic": {"", Mauve}, "bmp": {"", Mauve}, "tiff": {"", Mauve}, "tif": {"", Mauve}, "svg": {"", Mauve}, "ico": {"", Mauve},
	"mp3": {"", Teal}, "wav": {"", Teal}, "flac": {"", Teal}, "m4a": {"", Teal}, "ogg": {"", Teal}, "aac": {"", Teal},
	"mp4": {"", Red}, "mov": {"", Red}, "mkv": {"", Red}, "avi": {"", Red}, "webm": {"", Red},
	"zip": {"", Yellow}, "tar": {"", Yellow}, "gz": {"", Yellow}, "tgz": {"", Yellow}, "bz2": {"", Yellow},
	"xz": {"", Yellow}, "7z": {"", Yellow}, "rar": {"", Yellow},
	"pdf": {"", Red}, "doc": {"", Blue}, "docx": {"", Blue}, "xls": {"", Green}, "xlsx": {"", Green}, "csv": {"", Green},
	"ppt": {"", Peach}, "pptx": {"", Peach}, "key": {"", Peach}, "pages": {"", Blue}, "numbers": {"", Green},
	"txt": {glyphText, Plain}, "text": {glyphText, Plain}, "rtf": {glyphText, Plain}, "log": {glyphText, Plain},
	"md": {"", Blue}, "markdown": {"", Blue},
	"go": {"", Teal}, "py": {"", Yellow}, "js": {"", Yellow}, "mjs": {"", Yellow}, "jsx": {"", Yellow},
	"ts": {"", Blue}, "tsx": {"", Blue}, "rs": {"", Peach}, "c": {"", Blue}, "h": {"", Blue},
	"cpp": {"", Blue}, "hpp": {"", Blue}, "java": {"", Red}, "kt": {"", Mauve}, "swift": {"", Peach},
	"rb": {"", Red}, "php": {"", Mauve}, "lua": {"", Blue}, "sql": {"", Teal}, "tf": {"", Mauve},
	"sh": {"", Green}, "zsh": {"", Green}, "bash": {"", Green}, "fish": {"", Green},
	"html": {"", Peach}, "htm": {"", Peach}, "css": {"", Blue}, "json": {"", Yellow}, "xml": {"", Peach},
	"yaml": {"", Yellow}, "yml": {"", Yellow}, "toml": {"", Yellow}, "ini": {"", Yellow}, "conf": {"", Yellow},
	"lock": {"", Yellow}, "sum": {"", Yellow}, "mod": {"", Teal},
	"dmg": {"", Yellow}, "iso": {"", Yellow}, "pkg": {"", Peach},
	"ttf": {"", Plain}, "otf": {"", Plain}, "woff": {"", Plain}, "woff2": {"", Plain},
}

// byName covers files known by their whole name.
var byName = map[string]kind{
	"dockerfile": {"", Blue}, "makefile": {"", Peach}, ".gitignore": {"", Peach}, ".gitattributes": {"", Peach},
	"license": {glyphText, Yellow}, "readme": {"", Blue},
}

func nerd(it Item, ext string) Icon {
	switch {
	case it.IsLink:
		return Icon{Glyph: glyphLink, Role: Teal}
	case it.IsDir && ext == "app":
		return Icon{Glyph: glyphApp, Role: Peach}
	case it.IsDir && it.Open:
		return Icon{Glyph: glyphFolderOpen, Role: Folder}
	case it.IsDir:
		return Icon{Glyph: glyphFolder, Role: Folder}
	}
	if k, ok := kindOf(it.Name); ok {
		g := kindGlyphs[k]
		return Icon{Glyph: g.glyph, RGB: g.rgb, HasRGB: true}
	}
	if k, ok := byName[strings.ToLower(it.Name)]; ok {
		return Icon{Glyph: k.glyph, Role: k.role}
	}
	if k, ok := byExt[ext]; ok {
		return Icon{Glyph: k.glyph, Role: k.role}
	}
	g := kindGlyphs[kFile]
	return Icon{Glyph: g.glyph, RGB: g.rgb, HasRGB: true}
}
