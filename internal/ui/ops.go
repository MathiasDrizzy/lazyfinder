package ui

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
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
	promptPlace // renaming a favorite of the sidebar
)

// promptState is the one-line question on the bottom row: a name to type, or a yes/no.
type promptState struct {
	kind   promptKind
	input  string
	target string   // the path being renamed, or the folder a new file or folder goes in
	paths  []string // the paths a trash confirmation is about
}

// dirOr is the folder the prompt was started for, or fallback when it has none.
func (p promptState) dirOr(fallback string) string {
	if p.target != "" {
		return p.target
	}
	return fallback
}

type clipboard struct {
	paths []string
	cut   bool
}

// subject names what an operation worked on: the file name, or how many items.
func subject(paths []string) string {
	if len(paths) == 1 {
		return filepath.Base(paths[0])
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
	case keymap.Duplicate:
		return m.duplicate(), true
	case keymap.Rename:
		return m.startRename(), true
	case keymap.NewFolder:
		return m.startNewFolder(), true
	case keymap.NewFile:
		return m.startNewFile(), true
	case keymap.Trash:
		return m.startTrash(), true
	case keymap.Undo:
		return m.undo(), true
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
	var made []string
	var err error
	if m.clip.cut {
		made, err = m.ops.Move(m.clip.paths, m.dir())
	} else {
		made, err = m.ops.Copy(m.clip.paths, m.dir())
	}
	if err != nil {
		m.setError(err)
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("Pegado: %s", "Pasted %s"), subject(m.clip.paths)))
	}
	if m.clip.cut && err == nil {
		m.clip = clipboard{}
	}
	return m.refresh(last(made))
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
	made, err := m.ops.Copy(ps, filepath.Dir(ps[0]))
	if err != nil {
		m.setError(err)
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("Duplicado: %s", "Duplicated %s"), subject(ps)))
	}
	return m.refresh(last(made))
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
	if !m.canWrite() {
		return nil
	}
	dir := m.createDir()
	m.prompt = promptState{kind: promptNewFolder, target: dir, input: fileops.UniqueName(dir, i18n.T("carpeta sin título", "untitled folder"))}
	return nil
}

func (m *Model) startNewFile() tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	dir := m.createDir()
	m.prompt = promptState{kind: promptNewFile, target: dir, input: fileops.UniqueName(dir, i18n.T("archivo sin título", "untitled file"))}
	return nil
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

func (m *Model) trash(ps []string) tea.Cmd {
	if err := m.ops.Trash(ps); err != nil {
		m.setError(err)
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("%s enviado a la Papelera", "Moved %s to the Trash"), subject(ps)))
	}
	return m.refresh("")
}

func (m *Model) undo() tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	if !m.ops.CanUndo() {
		m.setStatus(i18n.T("No hay nada que deshacer", "Nothing to undo"))
		return nil
	}
	if _, err := m.ops.Undo(); err != nil {
		m.setError(err)
	} else {
		m.setStatus(i18n.T("Deshecho", "Undone"))
	}
	return m.refresh("")
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
	if m.prompt.kind == promptFilter {
		return m.filterKey(k)
	}
	switch key {
	case "esc":
		m.prompt = promptState{}
	case "enter":
		return m.promptSubmit()
	case "backspace":
		if r := []rune(m.prompt.input); len(r) > 0 {
			m.prompt.input = string(r[:len(r)-1])
		}
	case "ctrl+u":
		m.prompt.input = ""
	default:
		if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
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
	case promptNewFolder:
		made, err = m.ops.Mkdir(p.dirOr(m.dir()), p.input)
	case promptNewFile:
		made, err = m.ops.Touch(p.dirOr(m.dir()), p.input)
	case promptTag:
		m.prompt = promptState{}
		return m.toggleTag(p.input)
	case promptPlace:
		m.prompt = promptState{}
		return m.renameFavorite(p.target, p.input)
	}
	if err != nil {
		m.setError(err) // the prompt stays open, so the name can be fixed
		return nil
	}
	m.prompt = promptState{}
	if p.kind == promptRename {
		m.moveTags(p.target, made)
		m.setStatus(fmt.Sprintf(i18n.T("Renombrado a %s", "Renamed to %s"), filepath.Base(made)))
	} else if p.kind == promptNewFile {
		m.setStatus(fmt.Sprintf(i18n.T("Archivo creado: %s", "Created file %s"), filepath.Base(made)))
	} else {
		m.setStatus(fmt.Sprintf(i18n.T("Carpeta creada: %s", "Created folder %s"), filepath.Base(made)))
	}
	return m.refresh(made)
}

func (m *Model) promptBar() string {
	switch m.prompt.kind {
	case promptRename:
		return fit(m.styles.accent(" "+i18n.T("Renombrar: ", "Rename: "))+m.prompt.input+"▏", m.w)
	case promptNewFolder:
		return fit(m.styles.accent(" "+i18n.T("Carpeta nueva: ", "New folder: "))+m.prompt.input+"▏", m.w)
	case promptNewFile:
		return fit(m.styles.accent(" "+i18n.T("Archivo nuevo: ", "New file: "))+m.prompt.input+"▏", m.w)
	case promptTag:
		return fit(m.styles.accent(" "+i18n.T("Etiqueta: ", "Tag: "))+m.prompt.input+"▏", m.w)
	case promptPlace:
		return fit(m.styles.accent(" "+i18n.T("Nombre del favorito: ", "Favorite name: "))+m.prompt.input+"▏", m.w)
	case promptFilter:
		return fit(m.styles.accent(" "+i18n.T("Filtro: ", "Filter: "))+m.prompt.input+"▏", m.w)
	}
	return fit(m.styles.accent(" "+fmt.Sprintf(i18n.T("¿Enviar %s a la Papelera? (s/n)", "Move %s to the Trash? (y/n)"), subject(m.prompt.paths))), m.w)
}
