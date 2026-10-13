package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/fileops"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

const infoInner = 54

// infoState is the Get Info popup.
type infoState struct {
	open bool
	d    fileops.Details
	kind string
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
	m.info = infoState{open: true, d: d, kind: m.cell(e, fsmodel.ColKind)}
	return nil
}

func (m *Model) infoGeom() (x, y, h int) {
	h = len(m.infoLines()) + 4
	return (m.w - infoInner - 4) / 2, max(1, (m.h-h)/2), h
}

func (m *Model) infoLines() []string {
	d := m.info.d
	row := func(label, value string) string {
		return fmt.Sprintf("%-12s", label) + value
	}
	lines := []string{row(i18n.T("Nombre", "Name"), d.Name), row(i18n.T("Clase", "Kind"), m.info.kind)}
	if d.Link != "" {
		lines = append(lines, row(i18n.T("Apunta a", "Points to"), d.Link))
	}
	lines = append(lines, row(i18n.T("Ubicación", "Where"), d.Where))
	if d.IsDir {
		text := fmt.Sprintf(i18n.T("%d archivos, %s", "%d files, %s"), d.Files, fsmodel.HumanSize(d.Total))
		if d.Files == 1 {
			text = fmt.Sprintf(i18n.T("1 archivo, %s", "1 file, %s"), fsmodel.HumanSize(d.Total))
		}
		lines = append(lines, row(i18n.T("Contenido", "Contents"), text))
	} else {
		lines = append(lines, row(i18n.T("Tamaño", "Size"), fsmodel.HumanSize(d.Size)))
	}
	lines = append(lines,
		row(i18n.T("Modificado", "Modified"), d.Modified.Format("2006-01-02 15:04")),
		row(i18n.T("Permisos", "Permissions"), d.Mode.String()))
	return lines
}

func (m *Model) infoBox() (x, y int, lines []string) {
	x, y, _ = m.infoGeom()
	content := m.infoLines()
	content = append(content, "", m.styles.dim(i18n.T("Cualquier tecla cierra", "Any key closes")))
	return x, y, m.box(i18n.T("Información", "Get Info"), content, infoInner)
}
