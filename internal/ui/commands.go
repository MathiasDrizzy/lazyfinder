package ui

import (
	"errors"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
)

// showCommands lists the custom commands from config.json ("commands"). Choosing one runs it on the selected row
// (as {file}) in the current folder (as {dir}); it is never run through a shell.
func (m *Model) showCommands() tea.Cmd {
	names := m.deps.Cfg.CommandNames()
	if len(names) == 0 {
		m.setStatus(i18n.T(`No hay comandos propios: defínelos en "commands" de config.json`, `No custom commands: define them under "commands" in config.json`))
		return nil
	}
	items := make([]pickItem, len(names))
	for i, n := range names {
		name, spec := n, m.deps.Cfg.Commands[n]
		items[i] = pickItem{label: name, hint: strings.Join(spec.Argv, " "), run: func() tea.Cmd { return m.runCustom(name, spec) }}
	}
	m.openPicker(i18n.T("Comandos", "Commands"), items)
	return nil
}

func (m *Model) runCustom(name string, spec config.Opener) tea.Cmd {
	file := ""
	if e, ok := m.selected(); ok {
		file = e.Path
	}
	if file == "" && slices.Contains(spec.Argv, config.FilePlaceholder) {
		m.setError(errors.New(i18n.T("Selecciona un archivo primero", "Select a file first")))
		return nil
	}
	return m.run(opener.Custom(name, spec, file, m.dir()))
}
