package ui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// redo does again what the last undo reversed.
func (m *Model) redo() tea.Cmd {
	if !m.canWrite() {
		return nil
	}
	if !m.ops.CanRedo() {
		m.setStatus(i18n.T("No hay nada que rehacer", "Nothing to redo"))
		return nil
	}
	if _, err := m.ops.Redo(); err != nil {
		m.setError(err)
	} else {
		m.setStatus(i18n.T("Rehecho", "Redone"))
	}
	return m.refresh("")
}

// showHistory lists what u can undo, the next one first, and then what U can redo. It only shows: the keys act.
func (m *Model) showHistory() tea.Cmd {
	h := m.ops.History()
	if len(h) == 0 {
		m.setStatus(i18n.T("No hay nada que deshacer ni rehacer", "Nothing to undo or redo"))
		return nil
	}
	items := make([]pickItem, len(h))
	for i, e := range h {
		hint := i18n.T("deshacer", "undo")
		if e.Redo {
			hint = i18n.T("rehacer", "redo")
		}
		items[i] = pickItem{label: kindWord(e.Kind) + " " + e.Subject, hint: hint, run: func() tea.Cmd { return nil }}
	}
	m.openPicker(i18n.T("Historial", "History"), items)
	return nil
}

// kindWord words an operation of the history in the user's language.
func kindWord(k fileops.Kind) string {
	switch k {
	case fileops.KindCopy:
		return i18n.T("Copia de", "Copy of")
	case fileops.KindMove:
		return i18n.T("Movida de", "Move of")
	case fileops.KindRename:
		return i18n.T("Renombrado de", "Rename of")
	case fileops.KindMkdir:
		return i18n.T("Nueva carpeta", "New folder")
	case fileops.KindTouch:
		return i18n.T("Archivo nuevo", "New file")
	case fileops.KindTrash:
		return i18n.T("Papelera de", "Trashing of")
	case fileops.KindChmod:
		return i18n.T("Permisos de", "Permissions of")
	}
	return string(k)
}
