package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
)

type promptKind int

const (
	promptNone promptKind = iota
	promptRename
	promptNewFolder
	promptNewFile
	promptTrash
	promptFilter
	promptTag
	promptPlace   // renaming a favorite of the sidebar
	promptConfirm // a yes/no question with its own answer (git operations that throw work away)
	promptChmod   // the permissions of paths, in octal
)

// promptState is the one-line question on the bottom row: a name to type, or a yes/no.
type promptState struct {
	kind   promptKind
	input  string
	target string   // the path being renamed, or the folder a new file or folder goes in
	made   string   // the new file or folder, made before its name is asked, like the Finder does
	whole  bool     // the name is selected entirely: the first letter typed replaces it
	paths  []string // the paths a trash confirmation is about
	ask    string   // the question of a promptConfirm
	yes    func() tea.Cmd
}

type clipboard struct {
	paths []string
	cut   bool
}

// subject names what an operation worked on: the file name, or how many items.
func subject(paths []string) string {
	if len(paths) == 1 {
		return shownName(filepath.Base(paths[0]))
	}
	return fmt.Sprintf(i18n.T("%d elementos", "%d items"), len(paths))
}

// targets are the paths an operation acts on: the marked rows, else the selected one.
func (m *Model) targets() []string {
	var out []string
	if len(m.marked) > 0 {
		for _, e := range m.entries() {
			if m.marked[e.Path] {
				out = append(out, e.Path)
			}
		}
		return out
	}
	if e, ok := m.selected(); ok {
		out = append(out, e.Path)
	}
	return out
}

func (m *Model) toggleMark(path string) {
	if m.marked[path] {
		delete(m.marked, path)
		return
	}
	m.marked[path] = true
}

func (m *Model) markCount() int { return len(m.marked) }

// canWrite refuses a change while the list shows search results, since there is no folder to refresh.
func (m *Model) canWrite() bool {
	if m.op.busy {
		m.setStatus(i18n.T("Espera a que termine la operación en curso (Esc la cancela)", "Wait for the running operation to end (Esc cancels it)"))
		return false
	}
	if m.searchOn() {
		m.setStatus(i18n.T("Cierra la búsqueda primero", "Close the search first"))
		return false
	}
	return true
}

// opsKey handles the file operation actions; it reports whether the action was one of them.
func (m *Model) opsKey(act keymap.Action) (tea.Cmd, bool) {
	switch act {
	case keymap.Mark:
		if e, ok := m.selected(); ok {
			m.toggleMark(e.Path)
			return m.setSelection(m.sel + 1), true
		}
		return nil, true
	case keymap.MarkAll:
		for _, e := range m.entries() {
			m.marked[e.Path] = true
		}
		return nil, true
	case keymap.Copy:
		return m.clip1(false), true
	case keymap.Cut:
		return m.clip1(true), true
	case keymap.Paste:
		return m.paste(), true
	case keymap.CopyPath:
		ps := m.targets()
		if len(ps) == 0 {
			return nil, true
		}
		return m.copyText(strings.Join(ps, "\n"), subject(ps)), true
	case keymap.Duplicate:
		return m.duplicate(), true
	case keymap.Rename:
		return m.startRename(), true
	case keymap.Chmod:
		return m.startChmod(), true
	case keymap.NewFolder:
		return m.startNewFolder(), true
	case keymap.NewFile:
		return m.startNewFile(), true
	case keymap.Trash:
		return m.startTrash(), true
	case keymap.Undo:
		return m.undo(), true
	case keymap.Redo:
		return m.redo(), true
	case keymap.History:
		return m.showHistory(), true
	case keymap.Info:
		return m.openInfo(), true
	}
	return nil, false
}

func (m *Model) clip1(cut bool) tea.Cmd {
	ps := m.targets()
	if len(ps) == 0 {
		return nil
	}
	m.clip = clipboard{paths: ps, cut: cut}
	if cut {
		m.setStatus(fmt.Sprintf(i18n.T("Cortado: %s", "Cut %s"), subject(ps)))
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("Copiado: %s", "Copied %s"), subject(ps)))
	}
	return nil
}

// refresh reloads the folder after a change and selects path when it is on screen.
func (m *Model) refresh(path string) tea.Cmd {
	clear(m.marked)
	cmd := m.reload()
	if path != "" {
		m.selectPath(path)
	}
	return tea.Batch(cmd, m.refreshPreview())
}

func (m *Model) paste() tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	if len(m.clip.paths) == 0 {
		m.setStatus(i18n.T("No hay nada que pegar", "Nothing to paste"))
		return nil
	}
	kind := opCopy
	if m.clip.cut {
		kind = opMove
	}
	return m.runOp(kind, m.clip.paths, m.dir())
}

func last(ps []string) string {
	if len(ps) == 0 {
		return ""
	}
	return ps[len(ps)-1]
}

func (m *Model) duplicate() tea.Cmd {
	ps := m.targets()
	if len(ps) == 0 || !m.canWrite() {
		return nil
	}
	return m.runOp(opDuplicate, ps, filepath.Dir(ps[0]))
}

// startChmod asks for the permissions of the marked rows, else of the selected one, starting from those of the first.
func (m *Model) startChmod() tea.Cmd {
	ps := m.targets()
	if len(ps) == 0 || !m.canWrite() {
		return nil
	}
	fi, err := os.Lstat(ps[0])
	if err != nil {
		m.setError(err)
		return nil
	}
	m.prompt = promptState{kind: promptChmod, input: fmt.Sprintf("%03o", fi.Mode().Perm()), paths: ps}
	return nil
}

func (m *Model) startRename() tea.Cmd {
	e, ok := m.selected()
	if !ok || !m.canWrite() {
		return nil
	}
	m.prompt = promptState{kind: promptRename, input: e.Name, target: e.Path}
	return nil
}

// createDir is the folder a new file or folder goes in: in the tree of the sidebar mode the selected folder (or the
// folder of the selected file), anywhere else the folder being shown.
func (m *Model) createDir() string {
	if e, ok := m.selected(); ok && m.deps.Follow {
		if e.IsDir {
			return e.Path
		}
		return filepath.Dir(e.Path)
	}
	return m.dir()
}

func (m *Model) startNewFolder() tea.Cmd {
	return m.startNew(promptNewFolder, i18n.T("carpeta sin título", "untitled folder"), m.ops.Mkdir)
}

func (m *Model) startNewFile() tea.Cmd {
	return m.startNew(promptNewFile, i18n.T("archivo sin título", "untitled file"), m.ops.Touch)
}

// startNew makes the new folder or file at once with a free default name and asks for its name on its own row, with
// the whole name selected, as the Finder does: what is typed replaces it, enter keeps it and esc takes it back.
func (m *Model) startNew(kind promptKind, base string, mk func(dir, name string) (string, error)) tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	dir := m.createDir()
	name := fileops.UniqueName(dir, base)
	made, err := mk(dir, name)
	if err != nil {
		m.setError(err)
		return nil
	}
	if dir != m.dir() {
		m.expanded[dir] = true // the new row shows inside its folder
	}
	cmd := m.refresh(made)
	m.prompt = promptState{kind: kind, target: dir, made: made, input: name, whole: true}
	return cmd
}

// cancelNew takes back the folder or file the prompt made, while it is still empty: the undo of its creation, with
// nothing left to redo.
func (m *Model) cancelNew() tea.Cmd {
	p := m.prompt
	m.prompt = promptState{}
	if p.made == "" {
		return nil
	}
	if _, err := m.ops.Retract(); err != nil {
		m.setError(err)
	}
	return m.refresh("")
}

func (m *Model) startTrash() tea.Cmd {
	ps := m.targets()
	if len(ps) == 0 || !m.canWrite() {
		return nil
	}
	if m.deps.Cfg.ConfirmDelete && len(ps) > m.deps.Cfg.Ops.ConfirmTrashAbove {
		m.prompt = promptState{kind: promptTrash, paths: ps}
		return nil
	}
	return m.trash(ps)
}

// trash runs apart from the UI like a paste: a Trash on another volume copies.
func (m *Model) trash(ps []string) tea.Cmd { return m.runOp(opTrash, ps, "") }

func (m *Model) undo() tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	if !m.ops.CanUndo() {
		m.setStatus(i18n.T("No hay nada que deshacer", "Nothing to undo"))
		return nil
	}
	return m.runOp(opUndo, nil, "") // apart from the UI: it may move as much back as a paste moved
}

// promptKey edits the prompt on the bottom row.
func (m *Model) promptKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	if m.prompt.kind == promptTrash {
		switch key {
		case "y", "Y", "s", "S", "enter":
			ps := m.prompt.paths
			m.prompt = promptState{}
			return m.trash(ps)
		case "n", "N", "esc":
			m.prompt = promptState{}
		}
		return nil
	}
	if m.prompt.kind == promptConfirm {
		switch key {
		case "y", "Y", "s", "S", "enter":
			yes := m.prompt.yes
			m.prompt = promptState{}
			return yes()
		case "n", "N", "esc":
			m.prompt = promptState{}
		}
		return nil
	}
	if m.prompt.kind == promptFilter {
		return m.filterKey(k)
	}
	whole := m.prompt.whole
	m.prompt.whole = false
	switch key {
	case "esc":
		if m.prompt.made != "" {
			return m.cancelNew()
		}
		m.prompt = promptState{}
	case "enter":
		return m.promptSubmit()
	case "backspace":
		if whole {
			m.prompt.input = ""
			break
		}
		if r := []rune(m.prompt.input); len(r) > 0 {
			m.prompt.input = string(r[:len(r)-1])
		}
	case "ctrl+u":
		m.prompt.input = ""
	default:
		if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			if whole {
				m.prompt.input = ""
			}
			m.prompt.input += k.Text
		}
	}
	return nil
}

func (m *Model) promptSubmit() tea.Cmd {
	p := m.prompt
	var made string
	var err error
	switch p.kind {
	case promptRename:
		made, err = m.ops.Rename(p.target, p.input)
	case promptNewFolder, promptNewFile:
		made = p.made
		if p.input != filepath.Base(p.made) {
			made, err = m.ops.Rename(p.made, p.input)
		}
	case promptTag:
		m.prompt = promptState{}
		return m.toggleTag(p.input)
	case promptPlace:
		m.prompt = promptState{}
		return m.renameFavorite(p.target, p.input)
	case promptChmod:
		return m.submitChmod(p)
	}
	if err != nil {
		m.setError(err) // the prompt stays open, so the name can be fixed
		return nil
	}
	m.prompt = promptState{}
	switch p.kind {
	case promptRename:
		m.moveTags(p.target, made)
		m.setStatus(fmt.Sprintf(i18n.T("Renombrado a %s", "Renamed to %s"), filepath.Base(made)))
	case promptNewFile:
		m.setStatus(fmt.Sprintf(i18n.T("Archivo creado: %s", "Created file %s"), filepath.Base(made)))
	default:
		m.setStatus(fmt.Sprintf(i18n.T("Carpeta creada: %s", "Created folder %s"), filepath.Base(made)))
	}
	return m.refresh(made)
}

func (m *Model) promptBar() string {
	switch m.prompt.kind {
	case promptRename:
		return fit(m.styles.accent(" "+i18n.T("Renombrar: ", "Rename: "))+m.prompt.input+"▏", m.w)
	case promptNewFolder:
		return fit(" "+m.newIcon(true)+m.styles.accent(i18n.T("Nueva carpeta: ", "New folder: "))+m.promptInput()+"▏", m.w)
	case promptNewFile:
		return fit(" "+m.newIcon(false)+m.styles.accent(i18n.T("Nuevo archivo: ", "New file: "))+m.promptInput()+"▏", m.w)
	case promptTag:
		return fit(m.styles.accent(" "+i18n.T("Etiqueta: ", "Tag: "))+m.prompt.input+"▏", m.w)
	case promptPlace:
		return fit(m.styles.accent(" "+i18n.T("Nombre del favorito: ", "Favorite name: "))+m.prompt.input+"▏", m.w)
	case promptFilter:
		return fit(m.styles.accent(" "+i18n.T("Filtro: ", "Filter: "))+m.prompt.input+"▏", m.w)
	case promptChmod:
		label := fmt.Sprintf(i18n.T("Permisos de %s (octal): ", "Permissions of %s (octal): "), subject(m.prompt.paths))
		return fit(m.styles.accent(" "+label)+m.prompt.input+"▏", m.w)
	case promptConfirm:
		return fit(m.styles.accent(" "+m.prompt.ask+i18n.T(" (s/n)", " (y/n)")), m.w)
	}
	return fit(m.styles.accent(" "+fmt.Sprintf(i18n.T("¿Enviar %s a la Papelera? (s/n)", "Move %s to the Trash? (y/n)"), subject(m.prompt.paths))), m.w)
}

// newIcon is the icon of a folder or a file that heads the prompt for a new one, so it is clear which is being made.
func (m *Model) newIcon(dir bool) string {
	if m.iconW() == 0 {
		if dir {
			return "▢ "
		}
		return "▤ "
	}
	return m.iconCell(m.icons.Pick(icons.Item{Name: "x", IsDir: dir}))
}

// promptInput is the text being typed, shown selected while the first letter typed would replace it.
func (m *Model) promptInput() string {
	if m.prompt.whole && m.prompt.input != "" {
		return m.styles.selected(m.prompt.input)
	}
	return m.prompt.input
}

// editing says whether the row of path shows its name being typed: a new folder or file, or a rename.
func (m *Model) editing(path string) bool {
	switch m.prompt.kind {
	case promptNewFolder, promptNewFile:
		return m.prompt.made == path
	case promptRename:
		return m.prompt.target == path
	}
	return false
}

// editedName is the name cell of the row being named: the text typed so far and the caret.
func (m *Model) editedName() string { return m.prompt.input + "▏" }

// submitChmod sets the permissions typed in the prompt; a mode that is not octal keeps the prompt open.
func (m *Model) submitChmod(p promptState) tea.Cmd {
	perm, err := fileops.ParseMode(p.input)
	if err == nil {
		err = m.ops.Chmod(p.paths, perm)
	}
	if err != nil {
		m.setError(err)
		return nil
	}
	m.prompt = promptState{}
	m.setStatus(fmt.Sprintf(i18n.T("Permisos de %s: %03o", "Permissions of %s set to %03o"), subject(p.paths), perm))
	return m.refresh(p.paths[0])
}
