package ui

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// gitContextItems are the lines the right-click menu of a changed file adds to Open with: what the Git tab does,
// for the window. A file with no changes gets none, and its right click opens Open with as before.
func (m *Model) gitContextItems(idx int, e fsmodel.Entry) []ctxItem {
	if !m.git.ok || e.IsDir {
		return nil
	}
	staged, unstaged := m.git.st.ChangesOf(e.Path)
	if len(staged)+len(unstaged) == 0 {
		return nil
	}
	items := []ctxItem{{i18n.T("Abrir con…", "Open with…"), func() tea.Cmd {
		m.focus, m.sel = focusList, idx
		return m.openMenu()
	}}}
	for _, c := range unstaged {
		items = append(items, ctxItem{i18n.T("Ver los cambios", "Show changes"), func() tea.Cmd { return m.openDiff(c, false) }})
	}
	for _, c := range staged {
		items = append(items, ctxItem{i18n.T("Ver los cambios preparados", "Show staged changes"), func() tea.Cmd { return m.openDiff(c, true) }})
	}
	if len(unstaged) > 0 {
		c := unstaged[0]
		items = append(items, ctxItem{i18n.T("Preparar", "Stage"), func() tea.Cmd {
			return m.gitRun(i18n.T("Preparando…", "Staging…"), i18n.T("Cambios preparados", "Changes staged"),
				func(root string) error { return git.Stage(root, c.Path) })
		}})
	}
	if len(staged) > 0 {
		c := staged[0]
		items = append(items, ctxItem{i18n.T("Quitar de lo preparado", "Unstage"), func() tea.Cmd {
			return m.gitRun(i18n.T("Quitando de lo preparado…", "Unstaging…"), i18n.T("Quitado de lo preparado", "Changes unstaged"),
				func(root string) error { return git.Unstage(root, c.Path) })
		}})
	}
	if len(unstaged) > 0 {
		c := unstaged[0]
		items = append(items, ctxItem{i18n.T("Descartar cambios", "Discard changes"), func() tea.Cmd {
			m.confirm(fmt.Sprintf(i18n.T("¿Descartar los cambios de %s?", "Discard the changes to %s?"), filepath.Base(c.Path)), func() tea.Cmd { return m.scmDiscard(c) })
			return nil
		}})
	}
	return items
}
