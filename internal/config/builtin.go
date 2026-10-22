package config

import "sort"

// builtin are the openers that ship with lazyfinder so "Open With" is useful before any configuration; Open With
// lists the ones that are installed. A user opener with the same name replaces one. They are not written to the
// default config file.
var builtin = map[string]Opener{
	"textedit": {Argv: []string{"open", "-a", "TextEdit", FilePlaceholder}, Mode: ModeGUI},
	"preview":  {Argv: []string{"open", "-a", "Preview", FilePlaceholder}, Mode: ModeGUI},
	"vscode":   {Argv: []string{"open", "-a", "Visual Studio Code", FilePlaceholder}, Mode: ModeGUI},
	"zed":      {Argv: []string{"open", "-a", "Zed", FilePlaceholder}, Mode: ModeGUI},
	"sublime":  {Argv: []string{"open", "-a", "Sublime Text", FilePlaceholder}, Mode: ModeGUI},
	"cursor":   {Argv: []string{"open", "-a", "Cursor", FilePlaceholder}, Mode: ModeGUI},
	"vim":      {Argv: []string{"vim", FilePlaceholder}, Mode: ModeTerminal},
	"nvim":     {Argv: []string{"nvim", FilePlaceholder}, Mode: ModeTerminal},
	"micro":    {Argv: []string{"micro", FilePlaceholder}, Mode: ModeTerminal},
	"nano":     {Argv: []string{"nano", FilePlaceholder}, Mode: ModeTerminal},
	"helix":    {Argv: []string{"hx", FilePlaceholder}, Mode: ModeTerminal},
	"emacs":    {Argv: []string{"emacs", "-nw", FilePlaceholder}, Mode: ModeTerminal},
}

// Opener returns the opener called name: the user's first, then the built-in one.
func (c Config) Opener(name string) (Opener, bool) {
	if op, ok := c.Openers[name]; ok {
		return op, true
	}
	op, ok := builtin[name]
	return op, ok
}

// BuiltinNames returns the names of the shipped openers that the user has not replaced, sorted.
func (c Config) BuiltinNames() []string {
	var out []string
	for n := range builtin {
		if _, replaced := c.Openers[n]; !replaced {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
