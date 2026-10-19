package ui

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const infoInner = 54

// infoState is the Get Info popup. A folder's contents are counted apart (counting until the count arrives).
type infoState struct {
	open     bool
	d        fileops.Details
	kind     string
	path     string
	counting bool
	id       int
	cancel   context.CancelFunc
}

// infoCountMsg is the count of a folder for the Get Info popup with the same id.
type infoCountMsg struct {
	id           int
	files, total int64
	err          error
}

func (m *Model) openInfo() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		return nil
	}
	d, err := fileops.Info(e.Path)
	if err != nil {
		m.setError(err)
		return nil
	}
	m.closeInfo()
	m.infoID++
	m.info = infoState{open: true, d: d, kind: m.cell(e, fsmodel.ColKind), path: e.Path, id: m.infoID}
	if !d.IsDir || d.Link != "" {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.info.counting, m.info.cancel = true, cancel
	id, path := m.infoID, e.Path
	return func() tea.Msg {
		files, total, err := fileops.Count(ctx, path)
		return infoCountMsg{id: id, files: files, total: total, err: err}
	}
}

// closeInfo closes the Get Info popup and stops the count of its folder.
func (m *Model) closeInfo() {
	if m.info.cancel != nil {
		m.info.cancel()
	}
	m.info = infoState{}
}

func (m *Model) onInfoCount(msg infoCountMsg) tea.Cmd {
	if !m.info.open || msg.id != m.info.id {
		return nil
	}
	m.info.counting = false
	m.info.d.Files, m.info.d.Total = msg.files, msg.total
	if msg.err != nil {
		m.setError(msg.err)
	}
	return nil
}

func (m *Model) infoGeom() (x, y, h int) {
	h = len(m.infoLines()) + 4
	return max(0, (m.w-min(infoInner, m.w-4)-4)/2), max(1, (m.h-h)/2), h
}

func (m *Model) infoLines() []string {
	d := m.info.d
	row := func(label, value string) string {
		return fmt.Sprintf("%-12s", label) + value
	}
	lines := []string{row(i18n.T("Nombre", "Name"), shownName(d.Name)), row(i18n.T("Clase", "Kind"), m.info.kind)}
	if d.Link != "" {
		lines = append(lines, row(i18n.T("Apunta a", "Points to"), d.Link))
	}
	lines = append(lines, row(i18n.T("Ubicación", "Where"), shownName(d.Where)))
	if d.IsDir && m.info.counting {
		lines = append(lines, row(i18n.T("Contenido", "Contents"), i18n.T("Contando…", "Counting…")))
	} else if d.IsDir {
		text := fmt.Sprintf(i18n.T("%d archivos, %s", "%d files, %s"), d.Files, fsmodel.HumanSize(d.Total))
		if d.Files == 1 {
			text = fmt.Sprintf(i18n.T("1 archivo, %s", "1 file, %s"), fsmodel.HumanSize(d.Total))
		}
		lines = append(lines, row(i18n.T("Contenido", "Contents"), text))
	} else {
		lines = append(lines, row(i18n.T("Tamaño", "Size"), fsmodel.HumanSize(d.Size)))
	}
	if ts := m.tags[m.info.path]; len(ts) > 0 {
		lines = append(lines, row(i18n.T("Etiquetas", "Tags"), strings.Join(ts, ", ")))
	}
	lines = append(lines,
		row(i18n.T("Modificado", "Modified"), d.Modified.Format("2006-01-02 15:04")),
		row(i18n.T("Permisos", "Permissions"), d.Mode.String()))
	if d.Owner != "" {
		lines = append(lines, row(i18n.T("Propietario", "Owner"), d.Owner))
	}
	return lines
}

func (m *Model) infoBox() (x, y int, lines []string) {
	x, y, _ = m.infoGeom()
	content := m.infoLines()
	content = append(content, "", m.styles.dim(i18n.T("Cualquier tecla cierra", "Any key closes")))
	return x, y, m.box(i18n.T("Información", "Get Info"), content, infoInner)
}
