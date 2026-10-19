package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

// settingsState is the settings popup. Every change applies at once and is saved to config.json (docs/adr/0008).
type settingsState struct {
	open bool
	sel  int
	top  int // first visible row when the window is too short for all of them
}

const (
	setTheme = iota
	setLanguage
	setHidden
	setFoldersFirst
	setScreenBackground
	setPopupBackground
	setIcons
	setKeys
	setDateFormat
	setDensity
	setColumns
	setColumnSeparators
	setSearchScope
	setSearchKinds
	setSearchBy
	setConfirmDelete
	setCdOnExit
	setHerdrDock
	setSidebarMode
	settingCount
)

const settingsInner = 54

func (m *Model) settingLabel(i int) string {
	switch i {
	case setTheme:
		return i18n.T("Tema", "Theme")
	case setLanguage:
		return i18n.T("Idioma", "Language")
	case setHidden:
		return i18n.T("Archivos ocultos", "Hidden files")
	case setFoldersFirst:
		return i18n.T("Carpetas primero", "Folders first")
	case setScreenBackground:
		return i18n.T("Fondo de pantalla", "Screen background")
	case setPopupBackground:
		return i18n.T("Fondo de popups", "Popup background")
	case setConfirmDelete:
		return i18n.T("Confirmar al borrar", "Confirm deletion")
	case setCdOnExit:
		return i18n.T("Quedarse en la carpeta al salir", "Stay in the folder on exit")
	case setHerdrDock:
		return i18n.T("Lateral en herdr", "Sidebar in herdr")
	case setSidebarMode:
		return i18n.T("Modo lateral (proyecto)", "Side mode (project)")
	case setIcons:
		return i18n.T("Iconos", "Icons")
	case setColumns:
		return i18n.T("Columnas", "Columns")
	case setColumnSeparators:
		return i18n.T("Separadores de columna", "Column separators")
	case setDateFormat:
		return i18n.T("Formato de fechas", "Date format")
	case setDensity:
		return i18n.T("Densidad de filas", "Row density")
	case setKeys:
		return i18n.T("Atajos", "Keybindings")
	case setSearchScope:
		return i18n.T("Buscar en", "Search in")
	case setSearchKinds:
		return i18n.T("Buscar", "Search for")
	}
	return i18n.T("Buscar por", "Search by")
}

func onOff(b bool) string {
	if b {
		return i18n.T("sí", "yes")
	}
	return "no" // lazymark leaves it as is in every language
}

func (m *Model) settingValue(i int) string {
	c := m.deps.Cfg
	switch i {
	case setTheme:
		return c.Theme
	case setLanguage:
		if c.Language == "auto" {
			return "auto · " + i18n.CurrentLanguage().Name() // like lazymark: what "auto" resolved to
		}
		l, _ := i18n.Parse(c.Language)
		return l.Name()
	case setHidden:
		return onOff(m.hidden)
	case setFoldersFirst:
		return onOff(m.sort.FoldersFirst)
	case setScreenBackground:
		if c.ScreenBackground == config.ScreenBackgroundTerminal {
			return i18n.T("terminal", "terminal")
		}
		return i18n.T("tema", "theme")
	case setPopupBackground:
		if c.PopupBackground == config.ScreenBackgroundTerminal {
			return i18n.T("terminal", "terminal")
		}
		return i18n.T("tema", "theme")
	case setConfirmDelete:
		return onOff(c.ConfirmDelete)
	case setCdOnExit:
		if m.shellLineMissing() {
			return onOff(c.CdOnExit) + " · " + i18n.T("falta shell", "needs shell")
		}
		return onOff(c.CdOnExit)
	case setHerdrDock:
		if c.Herdr.Dock == "right" {
			return i18n.T("derecha", "right")
		}
		return i18n.T("izquierda", "left")
	case setSidebarMode:
		return onOff(m.projectMode())
	case setIcons:
		return c.Icons
	case setColumns:
		return columnPresetName(config.ColumnPreset(c.Columns))
	case setColumnSeparators:
		return onOff(c.ColumnSeparators)
	case setDateFormat:
		return dateFormatName(c.DateFormat)
	case setDensity:
		return densityName(c.Density)
	case setKeys:
		if c.KeybindingMode == keymap.ModeLazy {
			return "Lazy"
		}
		return "Lazy + Vim (hjkl)"
	case setSearchScope:
		return scopeName(orDefault(c.Search.Scope, searchScopes))
	case setSearchKinds:
		return kindsName(orDefault(c.Search.Kinds, searchKinds))
	}
	return byName(orDefault(c.Search.By, searchBys))
}

// cycle returns the item after (dir 1) or before (dir -1) cur in list, wrapping around.
func cycle(list []string, cur string, dir int) string {
	i := slices.Index(list, cur)
	return list[((i+dir)%len(list)+len(list))%len(list)]
}

func languageCodes() []string {
	codes := []string{"auto"}
	for _, l := range i18n.Languages {
		codes = append(codes, string(l))
	}
	return codes
}

// change moves setting i one step and saves it.
func (m *Model) change(i, dir int) tea.Cmd {
	var key string
	var val any
	var cmd tea.Cmd
	switch i {
	case setTheme:
		m.deps.Cfg.Theme = cycle(theme.Names(), m.deps.Cfg.Theme, dir)
		key, val = "theme", m.deps.Cfg.Theme
		m.restyle()
	case setLanguage:
		m.deps.Cfg.Language = cycle(languageCodes(), m.deps.Cfg.Language, dir)
		key, val = "language", m.deps.Cfg.Language
		i18n.SetLanguage(m.deps.Cfg.Language)
		cmd = m.reload() // the kinds of the files are written in the language of the read
	case setHidden:
		m.hidden = !m.hidden
		key, val = "show_hidden", m.hidden
		m.deps.Cfg.ShowHidden = m.hidden
		cmd = m.reload()
	case setFoldersFirst:
		s := m.sort
		s.FoldersFirst = !s.FoldersFirst
		key, val = "sort.folders_first", s.FoldersFirst
		m.deps.Cfg.Sort.FoldersFirst = s.FoldersFirst
		cmd = m.resort(s)
	case setScreenBackground:
		m.deps.Cfg.ScreenBackground = cycle([]string{config.ScreenBackgroundTheme, config.ScreenBackgroundTerminal}, m.deps.Cfg.ScreenBackground, dir)
		key, val = "screen_background", m.deps.Cfg.ScreenBackground
		m.restyle()
	case setPopupBackground:
		m.deps.Cfg.PopupBackground = cycle([]string{config.ScreenBackgroundTheme, config.ScreenBackgroundTerminal}, m.deps.Cfg.PopupBackground, dir)
		key, val = "popup_background", m.deps.Cfg.PopupBackground
	case setConfirmDelete:
		m.deps.Cfg.ConfirmDelete = !m.deps.Cfg.ConfirmDelete
		key, val = "confirm_delete", m.deps.Cfg.ConfirmDelete
	case setCdOnExit:
		m.deps.Cfg.CdOnExit = !m.deps.Cfg.CdOnExit
		key, val = "cd_on_exit", m.deps.Cfg.CdOnExit
	case setHerdrDock:
		m.deps.Cfg.Herdr.Dock = cycle([]string{"left", "right"}, m.deps.Cfg.Herdr.Dock, dir)
		key, val = "herdr.dock", m.deps.Cfg.Herdr.Dock
	case setSidebarMode:
		m.toggleProject()
		return nil // a mode of this run, not a saved setting
	case setIcons:
		m.deps.Cfg.Icons = cycle([]string{"nerd", "emoji", "ascii", "off"}, m.deps.Cfg.Icons, dir)
		key, val = "icons", m.deps.Cfg.Icons
		m.icons.Style = icons.Style(m.deps.Cfg.Icons)
	case setColumns:
		next := "basic"
		if config.ColumnPreset(m.deps.Cfg.Columns) == "basic" {
			next = "full"
		}
		m.applyColumnPreset(next)
		return nil // applyColumnPreset saved it
	case setColumnSeparators:
		m.deps.Cfg.ColumnSeparators = !m.deps.Cfg.ColumnSeparators
		key, val = "column_separators", m.deps.Cfg.ColumnSeparators
	case setDateFormat:
		m.deps.Cfg.DateFormat = cycle(config.DateFormats, m.deps.Cfg.DateFormat, dir)
		key, val = "date_format", m.deps.Cfg.DateFormat
		m.ensureVisible()
	case setDensity:
		m.deps.Cfg.Density = cycle(config.Densities, m.deps.Cfg.Density, dir)
		key, val = "density", m.deps.Cfg.Density
		m.ensureVisible()
	case setKeys:
		m.deps.Cfg.KeybindingMode = cycle([]string{keymap.ModeDual, keymap.ModeLazy}, m.deps.Cfg.KeybindingMode, dir)
		key, val = "keybinding_mode", m.deps.Cfg.KeybindingMode
		m.keys, _ = keymap.New(keymap.WithMode(m.deps.Cfg.Keys, m.deps.Cfg.KeybindingMode))
	case setSearchScope, setSearchKinds, setSearchBy:
		what := map[int]string{setSearchScope: "scope", setSearchKinds: "kinds", setSearchBy: "by"}[i]
		m.deps.Cfg.Search.Scope = orDefault(m.deps.Cfg.Search.Scope, searchScopes)
		m.deps.Cfg.Search.Kinds = orDefault(m.deps.Cfg.Search.Kinds, searchKinds)
		m.deps.Cfg.Search.By = orDefault(m.deps.Cfg.Search.By, searchBys)
		switch what {
		case "scope":
			m.deps.Cfg.Search.Scope = cycle(searchScopes, m.deps.Cfg.Search.Scope, dir)
		case "kinds":
			m.deps.Cfg.Search.Kinds = cycle(searchKinds, m.deps.Cfg.Search.Kinds, dir)
		default:
			m.deps.Cfg.Search.By = cycle(searchBys, m.deps.Cfg.Search.By, dir)
		}
		key = "search." + what
		val = map[string]string{"scope": m.deps.Cfg.Search.Scope, "kinds": m.deps.Cfg.Search.Kinds, "by": m.deps.Cfg.Search.By}[what]
	}
	if m.deps.ConfigPath != "" {
		if err := config.Update(m.deps.ConfigPath, map[string]any{key: val}); err != nil {
			m.setError(err)
		}
	}
	return cmd
}

func (m *Model) settingsKey(key string, act keymap.Action) tea.Cmd {
	switch {
	case act == keymap.Quit:
		return m.quit()
	case key == "esc" || act == keymap.Settings:
		m.set = settingsState{}
	case act == keymap.Up:
		m.set.sel = wrap(m.set.sel-1, settingCount)
	case act == keymap.Down:
		m.set.sel = wrap(m.set.sel+1, settingCount)
	case act == keymap.First:
		m.set.sel = 0
	case act == keymap.Last:
		m.set.sel = settingCount - 1
	case key == "left" || key == "h":
		return m.change(m.set.sel, -1)
	case key == "right" || key == "l" || key == "enter" || key == "space":
		return m.change(m.set.sel, 1)
	}
	return nil
}

// settingsView is the part of the rows that fits the window: inner width, visible rows and whether the hint line fits.
func (m *Model) settingsView() (inner, rows int, hint bool) {
	inner = min(settingsInner, max(8, m.w-4))
	rows = min(settingCount, max(1, m.h-4))
	hint = rows == settingCount && m.h >= settingCount+4
	if hint {
		rows = min(settingCount, m.h-6)
		hint = rows == settingCount
	}
	return inner, rows, hint
}

func (m *Model) settingsGeom() (x, y int) {
	inner, rows, hint := m.settingsView()
	h := rows + 2
	if hint {
		h += 2
	}
	return max(0, (m.w-inner-4)/2), max(0, (m.h-h)/2)
}

func (m *Model) settingsBox() (x, y int, lines []string) {
	x, y = m.settingsGeom()
	inner, rows, hint := m.settingsView()
	m.set.top = max(0, min(m.set.top, settingCount-rows))
	if m.set.sel < m.set.top {
		m.set.top = m.set.sel
	}
	if m.set.sel >= m.set.top+rows {
		m.set.top = m.set.sel - rows + 1
	}
	content := make([]string, 0, rows+2)
	for i := m.set.top; i < m.set.top+rows; i++ {
		label, value := m.settingLabel(i), "‹ "+m.settingValue(i)+" ›"
		if vmax := max(8, inner-3-min(ansi.StringWidth(label), 11)); ansi.StringWidth(value) > vmax { // narrow window: both give way
			value = "‹ " + ansi.Truncate(m.settingValue(i), vmax-4, "…") + " ›"
		}
		pad := max(1, inner-ansi.StringWidth(label)-ansi.StringWidth(value)-2)
		line := "  " + label + spaces(pad) + value
		if ansi.StringWidth(line) > inner { // narrow window: the label gives way to the value
			label = ansi.Truncate(label, max(1, inner-ansi.StringWidth(value)-3), "…")
			line = "  " + label + " " + value
		}
		if i == m.set.sel {
			line = m.styles.selected(ansi.Strip(fit(line, inner)))
		}
		content = append(content, line)
	}
	switch {
	case hint && m.set.sel == setCdOnExit && m.shellLineMissing():
		line := m.deps.ShellSetup.Line
		content = append(content, m.styles.dim(fmt.Sprintf(i18n.T("Agrega a %s:", "Add to %s:"), m.deps.ShellSetup.File)),
			ansi.Truncate(line, inner, "…"))
	case hint:
		text := i18n.T("←/→ cambiar · clic: seleccionar, 2.º clic: cambiar", "←/→ change · click: select, 2nd click: change")
		if ansi.StringWidth(text) > inner {
			text = i18n.T("←/→ cambiar · Esc cierra", "←/→ change · Esc closes")
		}
		content = append(content, "", m.styles.dim(text))
	}
	title := i18n.T("Ajustes", "Settings")
	if rows < settingCount { // a scrolled list says where it is
		if counter := fmt.Sprintf("%d/%d", m.set.sel+1, settingCount); inner-4 >= ansi.StringWidth(title+" "+counter) {
			title += " " + counter
		} else {
			title = counter
		}
	}
	return x, y, m.box(title, content, inner)
}

// shellLineMissing says cd_on_exit is on but cannot work: lazyfinder was not started by the shell function.
func (m *Model) shellLineMissing() bool {
	return m.deps.Cfg.CdOnExit && m.deps.ShellSetup.Line != ""
}

func spaces(n int) string { return strings.Repeat(" ", n) }

func (m *Model) settingsClick(x, y int) tea.Cmd {
	mx, my, bx := m.settingsBox()
	if !inBox(x, y, mx, my, bx) {
		return nil // like lazymark: a popup closes only with Esc
	}
	_, rows, _ := m.settingsView()
	if i := m.set.top + y - my - 1; y-my-1 >= 0 && y-my-1 < rows && i < settingCount {
		if m.set.sel != i { // the first click selects, the second changes
			m.set.sel = i
			return nil
		}
		return m.change(i, 1)
	}
	return nil
}

// dateFormatName and densityName are the words Settings shows for the config values (the file keeps the keys).
func dateFormatName(f string) string {
	switch f {
	case "absolute":
		return i18n.T("absoluto", "absolute")
	case "relative":
		return i18n.T("relativo", "relative")
	case "numeric":
		return i18n.T("numérico", "numeric")
	}
	return f
}

func densityName(d string) string {
	switch d {
	case config.DensityNormal:
		return i18n.T("normal", "normal")
	case config.DensityComfortable:
		return i18n.T("amplia", "wide")
	}
	return i18n.T("compactas", "compact")
}

func columnPresetName(p string) string {
	switch p {
	case "basic":
		return i18n.T("básico", "basic")
	case "full":
		return i18n.T("completo", "full")
	}
	return i18n.T("personalizado", "custom")
}
