package ui

import (
	tea "charm.land/bubbletea/v2"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// helpState is the `?` popup: every key and mouse gesture, with the keys the user has now.
type helpState struct {
	open bool
	top  int
}

type helpRow struct {
	keys string // already written for the screen
	what string
}

func (m *Model) keysOf(a keymap.Action) string {
	var out []string
	for _, k := range m.keys.Keys(a)[:min(2, len(m.keys.Keys(a)))] {
		out = append(out, keyLabel(k))
	}
	return strings.Join(out, " ")
}

// helpRows lists the help in sections; a section is a row with no keys.
func (m *Model) helpRows() []helpRow {
	k := m.keysOf
	return []helpRow{
		{"", i18n.T("Moverse", "Moving")},
		{k(keymap.Up) + " " + k(keymap.Down), i18n.T("Mover la selección", "Move the selection")},
		{k(keymap.Open) + " / " + k(keymap.Parent), i18n.T("Abrir / carpeta superior", "Open / parent folder")},
		{k(keymap.Back) + " " + k(keymap.Forward), i18n.T("Atrás / adelante en el historial", "Back / forward in history")},
		{k(keymap.Goto), i18n.T("Ir a una ruta", "Go to a path")},
		{"", i18n.T("Buscar", "Search")},
		{k(keymap.Search), i18n.T("Buscar; Tab, Ctrl+K y Ctrl+N cambian dónde, qué y cómo", "Search; Tab, Ctrl+K and Ctrl+N change where, what and how")},
		{"kind:image", i18n.T("En la búsqueda: kind: filtra por clase (image, text, folder…)", "In the search: kind: filters by kind (image, text, folder…)")},
		{"ext:go  name:report", i18n.T("ext: la extensión; name: una parte del nombre; tag: la etiqueta", "ext: the extension; name: part of the name; tag: the tag")},
		{"size:>10MB", i18n.T("size: el tamaño (> o <, con KB, MB o GB)", "size: the size (> or <, with KB, MB or GB)")},
		{"modified:week", i18n.T("modified: y created: la fecha (today, week, 2026-10-01…)", "modified: and created: the date (today, week, 2026-10-01…)")},
		{"in:invoice", i18n.T("in: busca dentro de los archivos", "in: searches inside the files")},
		{"", i18n.T("Favoritos y lugares", "Favorites and places")},
		{k(keymap.Bookmark), i18n.T("Añadir la carpeta a Favoritos (o quitarla)", "Add the folder to Favorites (or remove it)")},
		{k(keymap.SetMark) + " " + k(keymap.JumpMark), i18n.T("Marcar la carpeta / ir a una marca", "Mark the folder / go to a mark")},
		{k(keymap.Focus), i18n.T("Pasar a la barra lateral y volver", "Switch to the sidebar and back")},
		{"d  F2  K  J", i18n.T("En la barra lateral: quitar, renombrar, subir, bajar un favorito", "In the sidebar: remove, rename, move up, move down a favorite")},
		{i18n.T("clic derecho", "right click"), i18n.T("Menú de un favorito o de una carpeta", "Menu of a favorite or a folder")},
		{"", i18n.T("Tamaños", "Sizes")},
		{"< >", i18n.T("Barra lateral más estrecha / más ancha", "Narrower / wider sidebar")},
		{k(keymap.ColumnNarrow) + " " + k(keymap.ColumnWiden), i18n.T("Columna ordenada más estrecha / más ancha", "Narrower / wider sorted column")},
		{k(keymap.FavoritesShorter) + " " + k(keymap.FavoritesTaller), i18n.T("Cuadro de Favoritos más bajo / más alto (también arrastrando su borde)", "Shorter / taller Favorites box (or drag its border)")},
		{i18n.T("arrastrar", "drag"), i18n.T("Los bordes; doble clic los ajusta al contenido", "The borders; a double click fits the content")},
		{"", i18n.T("Archivos", "Files")},
		{k(keymap.Copy) + " " + k(keymap.Cut) + " " + k(keymap.Paste), i18n.T("Copiar, cortar, pegar", "Copy, cut, paste")},
		{k(keymap.CopyPath), i18n.T("Copiar la ruta al portapapeles", "Copy the path to the clipboard")},
		{k(keymap.Chmod), i18n.T("Cambiar los permisos (octal)", "Change the permissions (octal)")},
		{k(keymap.Rename) + "  " + k(keymap.Trash), i18n.T("Renombrar / enviar a la Papelera", "Rename / move to the Trash")},
		{k(keymap.NewFile) + "  " + k(keymap.NewFolder), i18n.T("Archivo nuevo / carpeta nueva", "New file / new folder")},
		{k(keymap.Undo), i18n.T("Deshacer", "Undo")},
		{k(keymap.Redo), i18n.T("Rehacer", "Redo")},
		{k(keymap.History), i18n.T("Historial de lo que se puede deshacer y rehacer", "History of what can be undone and redone")},
		{k(keymap.Preview) + " " + k(keymap.Info), i18n.T("Vista previa / información", "Preview / info")},
		{k(keymap.Sort) + " " + k(keymap.Hidden) + " " + k(keymap.Layout), i18n.T("Ordenar / ocultos / vista de columnas", "Sort / hidden / column view")},
		{k(keymap.Project), i18n.T("Modo proyecto: árbol con estado de git", "Project mode: a tree with git state")},
		{k(keymap.Settings) + " " + k(keymap.Help) + " " + k(keymap.Quit), i18n.T("Ajustes / esta ayuda / salir", "Settings / this help / quit")},
	}
}

const (
	helpInner  = 96
	helpKeysW  = 24 // the column of the keys when the popup is wide enough to hold two
	helpWideAt = 60 // under this inner width the text goes under its keys, as the herdr sidebar is that narrow
)

// helpInnerW is the width of the text of the popup.
func (m *Model) helpInnerW() int { return min(helpInner, m.w-6) }

// helpLines writes the help to the width of the popup: the keys and the text side by side when they fit, else the text
// under its keys. Whatever is long wraps, so no line is cut.
func (m *Model) helpLines() []string {
	iw := m.helpInnerW()
	wide := iw >= helpWideAt
	var out []string
	for _, r := range m.helpRows() {
		if r.keys == "" {
			out = append(out, m.styles.accent(ansi.Truncate(r.what, iw, "…")))
			continue
		}
		if !wide {
			out = append(out, "  "+ansi.Truncate(r.keys, max(1, iw-2), "…"))
			for _, l := range wrapText(r.what, max(1, iw-4)) {
				out = append(out, "    "+l)
			}
			continue
		}
		keys := ansi.Truncate(r.keys, helpKeysW-2, "…")
		for i, l := range wrapText(r.what, max(1, iw-helpKeysW)) {
			if i == 0 {
				out = append(out, "  "+keys+spaces(max(1, helpKeysW-2-ansi.StringWidth(keys)))+l)
				continue
			}
			out = append(out, spaces(helpKeysW)+l)
		}
	}
	return out
}

// wrapText breaks s into lines of at most w cells, at spaces.
func wrapText(s string, w int) []string {
	return strings.Split(ansi.Wrap(s, w, ""), "\n")
}

func (m *Model) helpRowsShown() int { return max(3, min(len(m.helpLines()), m.h-6)) }

func (m *Model) helpBox() (x, y int, lines []string) {
	all := m.helpLines()
	n := m.helpRowsShown()
	top := max(0, min(m.help.top, len(all)-n))
	iw := m.helpInnerW()
	content := all[top : top+n]
	x, y = (m.w-iw-4)/2, (m.h-n-2)/2
	return x, y, m.box(i18n.T("Ayuda", "Help"), content, iw)
}

func (m *Model) helpKey(key string, act keymap.Action) tea.Cmd {
	n := len(m.helpLines()) - m.helpRowsShown()
	switch {
	case act == keymap.Quit:
		return m.quit()
	case key == "esc" || act == keymap.Help:
		m.help = helpState{}
	case act == keymap.Up:
		m.help.top = max(0, m.help.top-1)
	case act == keymap.Down:
		m.help.top = min(max(0, n), m.help.top+1)
	}
	return nil
}
