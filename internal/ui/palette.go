package ui

import (
	"slices"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
)

// cursorActions only move the selection: the palette leaves them out, since a key does them better.
var cursorActions = []keymap.Action{keymap.Up, keymap.Down, keymap.PageUp, keymap.PageDown, keymap.First, keymap.Last}

// actionName words an action for the palette in the user's language.
func actionName(a keymap.Action) string {
	names := map[keymap.Action]string{
		keymap.Open: i18n.T("Abrir", "Open"), keymap.Parent: i18n.T("Carpeta superior", "Parent folder"),
		keymap.Back: i18n.T("Atrás", "Back"), keymap.Forward: i18n.T("Adelante", "Forward"),
		keymap.Preview: i18n.T("Vista previa", "Preview"), keymap.OpenWith: i18n.T("Abrir con…", "Open with…"),
		keymap.QuickLook: i18n.T("Vista rápida", "Quick Look"), keymap.Sort: i18n.T("Cambiar el orden", "Change the sort"),
		keymap.SortReverse: i18n.T("Invertir el orden", "Reverse the sort"),
		keymap.Hidden:      i18n.T("Mostrar u ocultar los archivos ocultos", "Show or hide hidden files"),
		keymap.Ignored:     i18n.T("Mostrar u ocultar lo ignorado por git", "Show or hide what git ignores"),
		keymap.Reload:      i18n.T("Recargar", "Reload"), keymap.Search: i18n.T("Buscar", "Search"),
		keymap.Goto: i18n.T("Ir a una ruta", "Go to a path"), keymap.Expand: i18n.T("Desplegar la carpeta", "Expand the folder"),
		keymap.Collapse: i18n.T("Plegar la carpeta", "Collapse the folder"), keymap.Mark: i18n.T("Marcar el ítem", "Mark the item"),
		keymap.MarkAll: i18n.T("Marcar todo", "Mark all"), keymap.Copy: i18n.T("Copiar", "Copy"), keymap.Cut: i18n.T("Cortar", "Cut"),
		keymap.Paste: i18n.T("Pegar", "Paste"), keymap.Duplicate: i18n.T("Duplicar", "Duplicate"),
		keymap.CopyPath: i18n.T("Copiar la ruta al portapapeles", "Copy the path to the clipboard"),
		keymap.Rename:   i18n.T("Renombrar", "Rename"), keymap.Chmod: i18n.T("Cambiar los permisos (octal)", "Change the permissions (octal)"),
		keymap.NewFolder: i18n.T("Nueva carpeta", "New folder"), keymap.NewFile: i18n.T("Archivo nuevo", "New file"),
		keymap.Trash: i18n.T("Enviar a la Papelera", "Move to the Trash"), keymap.Undo: i18n.T("Deshacer", "Undo"),
		keymap.Redo: i18n.T("Rehacer", "Redo"), keymap.History: i18n.T("Historial", "History"),
		keymap.Info: i18n.T("Información", "Get Info"), keymap.Layout: i18n.T("Vista de columnas", "Column view"),
		keymap.TagKey: i18n.T("Etiquetar", "Tag"), keymap.Tags: i18n.T("Etiquetas", "Tags"),
		keymap.SavedSearches: i18n.T("Búsquedas guardadas", "Saved searches"), keymap.Filter: i18n.T("Filtrar la carpeta", "Filter the folder"),
		keymap.Commands: i18n.T("Comandos propios", "Custom commands"), keymap.Recents: i18n.T("Recientes", "Recents"),
		keymap.Bookmark: i18n.T("Añadir la carpeta a Favoritos (o quitarla)", "Add the folder to Favorites (or remove it)"),
		keymap.SetMark:  i18n.T("Marcar la carpeta con una letra", "Mark the folder with a letter"),
		keymap.JumpMark: i18n.T("Ir a una marca", "Go to a mark"), keymap.NewTab: i18n.T("Pestaña nueva", "New tab"),
		keymap.CloseTab: i18n.T("Cerrar la pestaña", "Close the tab"), keymap.NextTab: i18n.T("Pestaña siguiente", "Next tab"),
		keymap.PrevTab:       i18n.T("Pestaña anterior", "Previous tab"),
		keymap.SidebarNarrow: i18n.T("Barra lateral más estrecha", "Narrower sidebar"),
		keymap.SidebarWiden:  i18n.T("Barra lateral más ancha", "Wider sidebar"),
		keymap.ColumnNarrow:  i18n.T("Columna más estrecha", "Narrower column"), keymap.ColumnWiden: i18n.T("Columna más ancha", "Wider column"),
		keymap.FavoritesShorter: i18n.T("Favoritos más bajo", "Shorter Favorites"),
		keymap.FavoritesTaller:  i18n.T("Favoritos más alto", "Taller Favorites"),
		keymap.Help:             i18n.T("Ayuda", "Help"), keymap.Focus: i18n.T("Pasar a la barra lateral y volver", "Switch to the sidebar and back"),
		keymap.Settings: i18n.T("Ajustes", "Settings"), keymap.Project: i18n.T("Modo proyecto: árbol con estado de git", "Project mode: a tree with git state"),
		keymap.Branch: i18n.T("Cambiar de rama de git", "Switch the git branch"), keymap.Quit: i18n.T("Salir", "Quit"),
		keymap.Palette: i18n.T("Paleta de comandos", "Command palette"),
	}
	if n, ok := names[a]; ok {
		return n
	}
	return string(a)
}

// showPalette lists every action with its keys; typing narrows the list and Enter runs the action.
func (m *Model) showPalette() tea.Cmd {
	var all []pickItem
	for _, a := range keymap.Actions {
		if a == keymap.Palette || slices.Contains(cursorActions, a) {
			continue
		}
		all = append(all, pickItem{label: actionName(a), hint: m.keysOf(a), match: string(a),
			run: func() tea.Cmd { return m.doAction(a) }})
	}
	m.picker = pickerState{open: true, title: i18n.T("Paleta de comandos", "Command palette"), items: all, all: all, filter: true}
	return nil
}

// doAction does what the action's key does in the list.
func (m *Model) doAction(a keymap.Action) tea.Cmd {
	if isResizeAction(a) {
		return m.resizeKey(a)
	}
	if m.scmOn() {
		key := ""
		if ks := m.keys.Keys(a); len(ks) > 0 {
			key = ks[0]
		}
		if cmd, ok := m.scmKey(key, a); ok {
			return cmd
		}
	}
	return m.listKey(a)
}

// paletteKey edits the query of the palette: what is typed narrows the list, the arrows move, Enter runs.
func (m *Model) paletteKey(k tea.KeyPressMsg) tea.Cmd {
	n := len(m.picker.items)
	switch key := k.String(); {
	case key == "esc":
		m.picker = pickerState{}
	case key == "enter":
		if n > 0 {
			return m.choosePick(m.picker.sel)
		}
	case key == "up" || key == "ctrl+k":
		m.picker.sel = wrap(m.picker.sel-1, max(n, 1))
	case key == "down" || key == "ctrl+j":
		m.picker.sel = wrap(m.picker.sel+1, max(n, 1))
	case key == "backspace":
		if _, size := utf8.DecodeLastRuneInString(m.picker.query); size > 0 {
			m.setQuery(m.picker.query[:len(m.picker.query)-size])
		}
	case k.Text != "":
		m.setQuery(m.picker.query + k.Text)
	}
	return nil
}

func (m *Model) setQuery(q string) {
	m.picker.query, m.picker.sel = q, 0
	m.picker.items = fuzzyFilter(m.picker.all, q)
}

// fuzzyFilter keeps the items whose label or match holds the letters of q in order, ignoring case: those where q
// appears whole come first, then those where its letters are closest together. Items that tie keep their order.
func fuzzyFilter(items []pickItem, q string) []pickItem {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return items
	}
	type scored struct {
		it    pickItem
		score int
	}
	var hits []scored
	for _, it := range items {
		best := -1
		for _, text := range []string{it.label, it.match} {
			if s := fuzzyScore(strings.ToLower(text), q); s >= 0 && (best < 0 || s < best) {
				best = s
			}
		}
		if best >= 0 {
			hits = append(hits, scored{it, best})
		}
	}
	slices.SortStableFunc(hits, func(a, b scored) int { return a.score - b.score })
	out := make([]pickItem, len(hits))
	for i, h := range hits {
		out[i] = h.it
	}
	return out
}

// fuzzyScore is -1 when text lacks the letters of q in order; otherwise lower is better: 0 when q appears whole,
// or one more than the letters skipped between the first and the last match.
func fuzzyScore(text, q string) int {
	if strings.Contains(text, q) {
		return 0
	}
	started, gaps, i := false, 0, 0
	for _, r := range text {
		if i == len(q) {
			break
		}
		qr, size := utf8.DecodeRuneInString(q[i:])
		if r != qr {
			if started {
				gaps++
			}
			continue
		}
		started = true
		i += size
	}
	if i < len(q) {
		return -1
	}
	return 1 + gaps
}
