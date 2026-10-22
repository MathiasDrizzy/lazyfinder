package ui

import (
	"fmt"
	"maps"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
	"github.com/charmbracelet/x/ansi"
)

// progState is the Programs popup of Settings: the programs Open With offers, the file types each one opens, and a
// last row to add one. Every change is saved to config.json at once ("openers" and "open_with").
type progState struct {
	open bool
	sel  int
	top  int
}

const progsInner = 60

// progRows is the number of rows: one per program and the row that adds one.
func (m *Model) progRows() int { return len(m.resolve.Programs()) + 1 }

// extsOf lists the file types that open with name, sorted.
func (m *Model) extsOf(name string) []string {
	var out []string
	for ext, n := range m.deps.Cfg.OpenWith {
		if n == name {
			out = append(out, ext)
		}
	}
	slices.Sort(out)
	return out
}

func (m *Model) progsKey(key string, act keymap.Action) tea.Cmd {
	names := m.resolve.Programs()
	n := len(names) + 1
	onProgram := m.progs.sel < len(names)
	switch {
	case key == "a" || key == "enter" && !onProgram:
		m.prompt = promptState{kind: promptProgram}
	case key == "enter":
		name := names[m.progs.sel]
		m.prompt = promptState{kind: promptProgramTypes, target: name, input: strings.Join(m.extsOf(name), ", "), whole: true}
	case key == "m" && onProgram:
		m.toggleProgramMode(names[m.progs.sel])
	case (key == "d" || key == "delete" || key == "backspace") && onProgram:
		m.removeProgram(names[m.progs.sel])
	case act == keymap.Quit:
		return m.quit()
	case key == "esc":
		m.progs = progState{}
	case act == keymap.Up:
		m.progs.sel = wrap(m.progs.sel-1, n)
	case act == keymap.Down:
		m.progs.sel = wrap(m.progs.sel+1, n)
	case act == keymap.First:
		m.progs.sel = 0
	case act == keymap.Last:
		m.progs.sel = n - 1
	}
	return nil
}

// addProgram adds the program typed in the prompt: an application by its name ("Sublime Text") opens in its own
// window with open -a, a command on PATH ("micro", "code --wait") in the terminal. The file goes last.
func (m *Model) addProgram(input string) error {
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return i18n.NewError("escribe un comando o el nombre de una aplicación", "type a command or the name of an application")
	}
	var name string
	var op config.Opener
	if app := strings.TrimSuffix(strings.TrimSpace(input), ".app"); opener.AppExists(app) {
		name, op = app, config.Opener{Argv: []string{"open", "-a", app, config.FilePlaceholder}, Mode: config.ModeGUI}
	} else if _, err := exec.LookPath(fields[0]); err == nil {
		argv := fields
		if !slices.Contains(argv, config.FilePlaceholder) {
			argv = append(argv, config.FilePlaceholder)
		}
		name, op = filepath.Base(fields[0]), config.Opener{Argv: argv, Mode: config.ModeTerminal}
	} else {
		return i18n.Errorf("%s no está instalado", "%s is not installed", fields[0])
	}
	name = programName(name)
	if _, ok := m.deps.Cfg.Openers[name]; ok {
		return i18n.Errorf("%s ya está en la lista", "%s is already in the list", name)
	}
	openers := maps.Clone(m.deps.Cfg.Openers)
	if openers == nil {
		openers = map[string]config.Opener{}
	}
	openers[name] = op
	m.saveOpeners(openers, m.deps.Cfg.OpenWith)
	m.progs.sel = max(0, slices.Index(m.resolve.Programs(), name))
	m.setStatus(fmt.Sprintf(i18n.T("Programa agregado: %s", "Program added: %s"), name))
	return nil
}

// programName is a name for the config file: lower case, and only letters, digits and dashes, since config keys are
// dotted.
func programName(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			return unicode.ToLower(r)
		}
		return '-'
	}, s)
	return strings.Trim(s, "-")
}

// setProgramTypes makes name the program of the file types typed ("md, .TXT go"), and of no other.
func (m *Model) setProgramTypes(name, input string) error {
	var exts []string
	for _, f := range strings.FieldsFunc(input, func(r rune) bool { return r == ',' || unicode.IsSpace(r) }) {
		ext := strings.ToLower(strings.TrimLeft(f, "."))
		if strings.ContainsFunc(ext, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '+'
		}) {
			return i18n.Errorf("%q no es un tipo de archivo (escribe extensiones: md, txt)", "%q is not a file type (type extensions: md, txt)", f)
		}
		if ext != "" && !slices.Contains(exts, ext) {
			exts = append(exts, ext)
		}
	}
	openWith := map[string]string{}
	for ext, n := range m.deps.Cfg.OpenWith {
		if n != name {
			openWith[ext] = n
		}
	}
	for _, ext := range exts {
		openWith[ext] = name
	}
	m.saveOpeners(m.deps.Cfg.Openers, openWith)
	m.setStatus(fmt.Sprintf(i18n.T("%s abre: %s", "%s opens: %s"), name, strings.Join(m.extsOf(name), ", ")))
	return nil
}

// toggleProgramMode switches a program between its own window and the terminal.
func (m *Model) toggleProgramMode(name string) {
	op, _ := m.deps.Cfg.Opener(name)
	op.Argv = slices.Clone(op.Argv)
	op.Mode = map[string]string{config.ModeGUI: config.ModeTerminal, config.ModeTerminal: config.ModeGUI}[op.Mode]
	openers := maps.Clone(m.deps.Cfg.Openers)
	if openers == nil {
		openers = map[string]config.Opener{}
	}
	openers[name] = op
	m.saveOpeners(openers, m.deps.Cfg.OpenWith)
}

// removeProgram takes a program off the list with the file types it opened; the ones lazyfinder ships stay.
func (m *Model) removeProgram(name string) {
	if _, ok := m.deps.Cfg.Openers[name]; !ok {
		m.setError(i18n.Errorf("%s viene con lazyfinder y no se quita", "%s ships with lazyfinder and cannot be removed", name))
		return
	}
	openers := maps.Clone(m.deps.Cfg.Openers)
	delete(openers, name)
	openWith := map[string]string{}
	for ext, n := range m.deps.Cfg.OpenWith {
		if n != name {
			openWith[ext] = n
		}
	}
	m.saveOpeners(openers, openWith)
	m.progs.sel = min(m.progs.sel, m.progRows()-1)
	m.setStatus(fmt.Sprintf(i18n.T("Programa quitado: %s", "Program removed: %s"), name))
}

// saveOpeners puts the programs and file types in effect and writes them to config.json.
func (m *Model) saveOpeners(openers map[string]config.Opener, openWith map[string]string) {
	if openers == nil {
		openers = map[string]config.Opener{}
	}
	if openWith == nil {
		openWith = map[string]string{}
	}
	m.deps.Cfg.Openers, m.deps.Cfg.OpenWith = openers, openWith
	m.resolve = opener.New(m.deps.Cfg)
	if m.deps.ConfigPath != "" {
		if err := config.Update(m.deps.ConfigPath, map[string]any{"openers": openers, "open_with": openWith}); err != nil {
			m.setError(err)
		}
	}
}

func (m *Model) modeName(mode string) string {
	if mode == config.ModeGUI {
		return i18n.T("ventana", "window")
	}
	return "terminal"
}

func (m *Model) progsView() (inner, rows int) {
	inner = min(progsInner, max(8, m.w-4))
	rows = min(m.progRows(), max(1, m.h-6))
	return inner, rows
}

func (m *Model) progsBox() (x, y int, lines []string) {
	inner, rows := m.progsView()
	names := m.resolve.Programs()
	n := len(names) + 1
	m.progs.sel = min(m.progs.sel, n-1)
	m.progs.top = max(0, min(m.progs.top, n-rows))
	if m.progs.sel < m.progs.top {
		m.progs.top = m.progs.sel
	}
	if m.progs.sel >= m.progs.top+rows {
		m.progs.top = m.progs.sel - rows + 1
	}
	content := make([]string, 0, rows+2)
	for i := m.progs.top; i < m.progs.top+rows; i++ {
		line := "  " + m.styles.accent(i18n.T("+ Agregar programa", "+ Add program"))
		if i < len(names) {
			op, _ := m.deps.Cfg.Opener(names[i])
			line = "  " + fit(names[i], 16) + m.styles.dim(fit(m.modeName(op.Mode), 10)) + strings.Join(m.extsOf(names[i]), ", ")
		}
		line = ansi.Truncate(line, inner, "…")
		if i == m.progs.sel {
			line = m.styles.selected(ansi.Strip(fit(line, inner)))
		}
		content = append(content, line)
	}
	if m.h >= rows+8 {
		hint := i18n.T("Enter tipos de archivo · a agregar · m modo · d quitar", "Enter file types · a add · m mode · d remove")
		content = append(content, "", m.styles.dim(ansi.Truncate(hint, inner, "…")))
	}
	h := len(content) + 2
	x, y = max(0, (m.w-inner-4)/2), max(0, (m.h-h)/2)
	return x, y, m.box(i18n.T("Programas", "Programs"), content, inner)
}

func (m *Model) progsClick(x, y int) tea.Cmd {
	mx, my, bx := m.progsBox()
	if !inBox(x, y, mx, my, bx) {
		return nil // like lazymark: a popup closes only with Esc
	}
	_, rows := m.progsView()
	if i := m.progs.top + y - my - 1; y-my-1 >= 0 && y-my-1 < rows && i < m.progRows() {
		if m.progs.sel != i { // the first click selects, the second opens it
			m.progs.sel = i
			return nil
		}
		return m.progsKey("enter", "")
	}
	return nil
}
