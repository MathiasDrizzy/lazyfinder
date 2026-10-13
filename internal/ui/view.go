package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/fsmodel"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/icons"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
	"github.com/charmbracelet/x/ansi"
)

const (
	sidebarWide   = 22
	sidebarNarrow = 18
	colGap        = 2 // cells between two columns; 3 with the │ separator
	prefixW       = 4 // pointer, space, folder marker, space
	gitGutterW    = 2 // git state and a space, only inside a repository
	minNameW      = 16
	crumbSep      = " › "
)

func columnLabel(c fsmodel.Column) string {
	switch c {
	case fsmodel.ColModified:
		return i18n.T("Fecha de modificación", "Date Modified")
	case fsmodel.ColCreated:
		return i18n.T("Fecha de creación", "Date Created")
	case fsmodel.ColSize:
		return i18n.T("Tamaño", "Size")
	case fsmodel.ColKind:
		return i18n.T("Clase", "Kind")
	case fsmodel.ColOpened:
		return i18n.T("Última apertura", "Date Last Opened")
	case fsmodel.ColExt:
		return i18n.T("Extensión", "Extension")
	case fsmodel.ColPerms:
		return i18n.T("Permisos", "Permissions")
	case fsmodel.ColTags:
		return i18n.T("Etiquetas", "Tags")
	case fsmodel.ColOwner:
		return i18n.T("Propietario", "Owner")
	case fsmodel.ColPath:
		return i18n.T("Ruta", "Path")
	}
	return i18n.T("Nombre", "Name")
}

// columnWidths are the widths a column has when nothing in config.json sets them and its content is shorter.
var columnWidths = map[fsmodel.Column]int{
	fsmodel.ColModified: 18, fsmodel.ColCreated: 18, fsmodel.ColOpened: 18, fsmodel.ColSize: 10, fsmodel.ColKind: 18,
	fsmodel.ColExt: 9, fsmodel.ColPerms: 10, fsmodel.ColTags: 12,
}

type colSpec struct {
	col  fsmodel.Column
	x, w int // x is relative to the list's left edge
}

func (m *Model) sidebarW() int {
	if m.deps.Follow {
		return 0
	}
	if w := m.deps.Cfg.SidebarWidth; w > 0 {
		return max(config.MinSidebar, min(w, m.w/2))
	}
	if m.w >= 90 {
		return sidebarWide
	}
	return sidebarNarrow
}

// The screen, top to bottom: toolbar (row 0), the two panels (rows 1..h-3), the breadcrumb (h-2) and the key bar (h-1).
// Each panel has a border, so the content of the list starts one cell in and its header sits on row 2.
func (m *Model) listX() int { return m.sidebarW() + 1 } // first content column of the list panel
func (m *Model) listW() int { return m.w - m.sidebarW() - 2 }

const (
	headerRow = 2 // column header
	firstRow  = 3 // first entry
)

// prefix is the width before the name: pointer, folder marker and, inside a repository, the git state.
func (m *Model) prefix() int {
	if m.git.ok {
		return prefixW + gitGutterW
	}
	return prefixW
}

func (m *Model) panelH() int { return m.h - 3 }

// dropOrder says which optional columns go first when the window is narrow.
var dropOrder = []fsmodel.Column{fsmodel.ColPath, fsmodel.ColTags, fsmodel.ColOwner, fsmodel.ColPerms, fsmodel.ColOpened, fsmodel.ColCreated, fsmodel.ColExt, fsmodel.ColKind, fsmodel.ColSize, fsmodel.ColModified}

// columns lays out the visible columns; optional ones are dropped by dropOrder until the name fits.
func (m *Model) columns() []colSpec {
	var extra []fsmodel.Column
	for _, c := range m.deps.Cfg.Columns {
		if col := fsmodel.Column(c); col != fsmodel.ColName {
			extra = append(extra, col)
		}
	}
	nameW := func() int {
		w := m.listW() - m.prefix()
		for _, c := range extra {
			w -= m.colWidth(c) + m.gap()
		}
		return w
	}
	for _, drop := range dropOrder {
		if nameW() >= minNameW {
			break
		}
		extra = slices.DeleteFunc(extra, func(c fsmodel.Column) bool { return c == drop })
	}
	specs := []colSpec{{col: fsmodel.ColName, x: m.prefix(), w: max(1, nameW())}}
	x := m.prefix() + specs[0].w + m.gap()
	for _, c := range extra {
		specs = append(specs, colSpec{col: c, x: x, w: m.colWidth(c)})
		x += m.colWidth(c) + m.gap()
	}
	return specs
}

func (m *Model) cell(e fsmodel.Entry, c fsmodel.Column) string {
	switch c {
	case fsmodel.ColModified:
		return m.date(e.Modified)
	case fsmodel.ColCreated:
		return m.date(e.Created)
	case fsmodel.ColSize:
		if e.IsDir {
			return "--"
		}
		return fsmodel.HumanSize(e.Size)
	case fsmodel.ColKind:
		return e.Kind
	case fsmodel.ColOpened:
		return m.date(e.Opened)
	case fsmodel.ColExt:
		return fsmodel.ExtOf(e)
	case fsmodel.ColPerms:
		return fsmodel.PermsOf(e)
	case fsmodel.ColTags:
		return strings.Join(e.Tags, ", ")
	case fsmodel.ColOwner:
		return e.Owner
	case fsmodel.ColPath:
		return m.homeShort(filepath.Dir(e.Path))
	}
	if m.searchOn() && m.search.deep() {
		if w := m.where(e); w != "" {
			return strings.Repeat("  ", e.Depth) + e.Name + "  ‹ " + w
		}
	}
	return strings.Repeat("  ", e.Depth) + e.Name
}

// place is a flattened sidebar line: a section title or a destination.
type place struct {
	title   string
	name    string
	path    string
	missing bool
	fav     int  // 1 + the index in sidebar.favorites for a favorite, else 0
	recents bool // the Recents entry: opens the list of recent folders instead of a folder
	divider bool // a line between two sections
}

func (m *Model) places() []place {
	var out []place
	isFav := false
	section := func(title string, ps []config.Place) {
		out = append(out, place{title: title})
		for i, p := range ps {
			_, err := os.Stat(p.Path)
			pl := place{name: localName(p.Name), path: p.Path, missing: err != nil}
			if isFav {
				pl.fav = i + 1
			}
			out = append(out, pl)
		}
	}
	isFav = true
	section(i18n.T("Favoritos", "Favorites"), m.deps.Cfg.Sidebar.Favorites)
	isFav = false
	out = slices.Insert(out, 1, place{name: i18n.T("Recientes", "Recents"), recents: true})
	out = append(out, place{divider: true})
	section(i18n.T("Ubicaciones", "Locations"), m.deps.Cfg.Sidebar.Locations)
	return out
}

// destinations returns the indexes into places() that can be selected.
func destinations(ps []place) []int {
	var idx []int
	for i, p := range ps {
		if p.title == "" && p.name != "" {
			idx = append(idx, i)
		}
	}
	return idx
}

type crumb struct {
	label, path string
	x, w        int
}

// crumbs lays out the breadcrumb for the bottom bar, dropping leading segments (shown as "…") when it is too long.
func (m *Model) crumbs() []crumb {
	dir := m.dir()
	var parts []crumb
	cur := dir
	for {
		if parent := filepath.Dir(cur); parent == cur {
			parts = append([]crumb{{label: "/", path: cur}}, parts...)
			break
		}
		parts = append([]crumb{{label: m.folderLabel(cur), path: cur}}, parts...)
		cur = filepath.Dir(cur)
	}
	width := func(ps []crumb, ellipsis bool) int {
		w := 1
		if ellipsis {
			w += 1 + ansi.StringWidth(crumbSep)
		}
		for i, p := range ps {
			if i > 0 {
				w += ansi.StringWidth(crumbSep)
			}
			w += ansi.StringWidth(p.label)
		}
		return w
	}
	ellipsis := false
	for len(parts) > 1 && width(parts, ellipsis) > m.w-1 {
		parts, ellipsis = parts[1:], true
	}
	x := 1
	if ellipsis {
		x += 1 + ansi.StringWidth(crumbSep)
	}
	for i := range parts {
		if i > 0 {
			x += ansi.StringWidth(crumbSep)
		}
		parts[i].x, parts[i].w = x, ansi.StringWidth(parts[i].label)
		x += parts[i].w
	}
	return parts
}

// paint puts the theme's background, and its text color for plain text, behind everything on the screen.
func (m *Model) paint(s string) string {
	if m.styles.text != nil {
		return theme.PaintBackground(s, m.styles.bg, m.styles.text)
	}
	return theme.PaintBackground(s, m.styles.bg)
}

// View renders the screen.
func (m *Model) View() tea.View {
	var content string
	if m.tooSmallNow() {
		content = m.tooSmall()
	} else {
		content = strings.Join(m.screen(), "\n")
		if m.styles.bg != nil && !m.popupsKeepTerminalBackground() {
			content = m.paint(content)
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	if m.mouse {
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.WindowTitle = "lazyfinder"
	return v
}

func (m *Model) tooSmall() string {
	lines := make([]string, max(m.h, 1))
	w := max(m.w, 1)
	lines[0] = ansi.Truncate(i18n.T("Terminal muy pequeña", "Terminal too small"), w, "")
	if len(lines) > 1 {
		lines[1] = ansi.Truncate(fmt.Sprintf(i18n.T("Hace falta %dx%d, hay %dx%d", "Need %dx%d, have %dx%d"), m.minSize()[0], m.minSize()[1], m.w, m.h), w, "")
	}
	return strings.Join(lines, "\n")
}

func (m *Model) screen() []string {
	lines, put := m.baseScreen()
	for _, p := range m.popupBoxes() {
		lines = put(lines, p.box, p.x, p.y)
	}
	return lines
}

// popupsKeepTerminalBackground says whether the screen is painted with the theme while popups are not
// (popup_background "terminal"): screen() then paints the base itself and View must not paint again.
func (m *Model) popupsKeepTerminalBackground() bool {
	return m.styles.bg != nil && m.deps.Cfg.PopupBackground == config.ScreenBackgroundTerminal
}

type popupBox struct {
	x, y int
	box  []string
}

// popupBoxes are the open popups, bottom first.
func (m *Model) popupBoxes() []popupBox {
	var out []popupBox
	add := func(open bool, f func() (int, int, []string)) {
		if open {
			x, y, bx := f()
			out = append(out, popupBox{x, y, bx})
		}
	}
	add(m.picker.open, m.pickerBox)
	add(m.menu.open, m.menuBox)
	add(m.pv.open, m.previewBox)
	add(m.set.open, m.settingsBox)
	add(m.info.open, m.infoBox)
	add(m.help.open, m.helpBox)
	add(m.ctx.open, m.ctxBox)
	return out
}

// baseScreen draws the panels and bars, and says how popups go on top of it: when the popups keep the terminal's
// background the base is painted here and each popup restores the theme background behind itself.
func (m *Model) baseScreen() ([]string, func([]string, []string, int, int) []string) {
	lines := make([]string, m.h)
	lines[0] = m.toolbar()
	list := m.panel(m.folderTitle(), m.counter(), m.listLines(), m.w-m.sidebarW(), m.panelH(), m.focus == focusList)
	if m.sidebarW() == 0 {
		for y := 0; y < m.panelH(); y++ {
			lines[y+1] = list[y]
		}
	} else {
		side := m.panel(i18n.T("Sitios", "Places"), "", m.sidebarLines(), m.sidebarW(), m.panelH(), m.focus == focusSidebar)
		for y := 0; y < m.panelH(); y++ {
			lines[y+1] = side[y] + list[y]
		}
	}
	lines[m.h-2] = m.breadcrumb()
	lines[m.h-1] = m.footerBar()
	if m.popupsKeepTerminalBackground() {
		lines = strings.Split(m.paint(strings.Join(lines, "\n")), "\n")
		after := theme.BackgroundSeq(m.styles.bg)
		return lines, func(base, box []string, x, y int) []string { return overlayAfter(base, box, x, y, after) }
	}
	return lines, overlay
}

func (m *Model) folderTitle() string {
	if m.dir() == "/" {
		return "/"
	}
	return m.folderLabel(m.dir())
}

// counter is the "n of m" in the bottom border of the list panel.
func (m *Model) counter() string {
	n := len(m.entries())
	if n == 0 {
		return fmt.Sprintf(i18n.T("%d de %d", "%d of %d"), 0, 0)
	}
	c := fmt.Sprintf(i18n.T("%d de %d", "%d of %d"), m.sel+1, n)
	if k := m.markCount(); k > 0 {
		c += fmt.Sprintf(i18n.T(" · %d seleccionados", " · %d selected"), k)
	}
	return c
}

const (
	gear = "⚙"
	lens = "⌕"
)

func (m *Model) toolbar() string {
	back, fwd := " ‹ ", " › "
	if !m.hist.CanBack() {
		back = m.styles.dim(back)
	}
	if !m.hist.CanForward() {
		fwd = m.styles.dim(fwd)
	}
	left := back + fwd + "  " + m.styles.bold(m.folderTitle())
	if m.filter != "" {
		left += "  " + m.styles.accent(i18n.T("filtro: ", "filter: ")+m.filter)
	}
	if l := m.tabLabel(); l != "" {
		left += "  " + m.styles.accent(l)
	}
	right := m.styles.dim(fmt.Sprintf(i18n.T("%d elementos", "%d items"), len(m.entries()))) + "  " + m.searchField() + "  " + m.gearButton() + " "
	if b := m.branchLabel(); b != "" {
		right = m.styles.accent(b) + "  " + right
	}
	pad := m.w - ansi.StringWidth(left) - ansi.StringWidth(right)
	if pad < 1 {
		return fit(left, m.w-m.gearW()) + "  " + m.gearButton() + " "
	}
	return left + strings.Repeat(" ", pad) + right
}

// searchFieldW is the width of the Search field at the top right, the one Finder has; narrow windows show just the lens.
const searchFieldW = 22

// fieldW is the cells the field takes, 1 (the lens) in a narrow window.
func (m *Model) fieldW() int {
	if m.w < 80 {
		return 1
	}
	return searchFieldW
}

// searchField draws the Search field: its hint while idle, the query while searching.
func (m *Model) searchField() string {
	w := m.fieldW()
	if w == 1 {
		return m.styles.accent(lens)
	}
	text := " " + lens + " " + i18n.T("Buscar", "Search")
	style := m.styles.selectedInactive
	if m.searchOn() {
		cursor := ""
		if m.search.mode == searchTyping {
			cursor = "▏"
		}
		text = " " + lens + " " + m.search.input + cursor
		style = m.styles.selected
	}
	return style(ansi.Strip(fit(text, w)))
}

func (m *Model) sidebarLines() []string {
	w := m.sidebarW() - 2
	rows := m.panelH() - 2
	out := make([]string, rows)
	ps := m.places()
	dests := destinations(ps)
	for i := range out {
		out[i] = strings.Repeat(" ", w)
		if i >= len(ps) {
			continue
		}
		p := ps[i]
		switch {
		case p.title != "":
			out[i] = m.styles.dim(fit(" "+p.title, w))
		case p.divider:
			out[i] = " " + m.styles.border(strings.Repeat("─", max(0, w-2))) + " "
		case p.name == "":
		default:
			glyph, lead := m.placeIcon(p)
			label := fit(lead+" "+p.name, w)
			if glyph != "" {
				label = fit(lead+glyph+" "+p.name, w)
			}
			focused := m.focus == focusSidebar && len(dests) > 0 && dests[min(m.sideSel, len(dests)-1)] == i
			switch {
			case focused:
				plain := "▸ " + p.name
				if glyph != "" {
					plain = "▸ " + ansi.Strip(glyph) + " " + p.name
				}
				label = m.styles.selected(ansi.Strip(fit(plain, w)))
			case p.missing:
				label = m.styles.dim(ansi.Strip(label))
			case p.path == m.dir():
				label = m.styles.bold(label)
			}
			out[i] = label
		}
	}
	return out
}

func (m *Model) listLines() []string {
	if m.columnsOn() {
		return m.millerLines()
	}
	rows := m.panelH() - 2
	out := make([]string, rows)
	specs := m.columns()
	hdr := strings.Repeat(" ", m.prefix())
	var parts []string
	for i, s := range specs {
		label := columnLabel(s.col)
		if s.col == m.sort.By {
			if m.sort.Descending {
				label += " ▼"
			} else {
				label += " ▲"
			}
		}
		if i > 0 {
			parts = append(parts, m.gapText())
		}
		parts = append(parts, m.styles.title(fit(label, s.w)))
	}
	out[0] = m.styles.title(hdr) + strings.Join(parts, "")
	out[0] += strings.Repeat(" ", max(0, m.listW()-ansi.StringWidth(out[0])))
	for r := 1; r < rows; r++ {
		idx := m.top + (r-1)/m.rowH()
		if (r-1)%m.rowH() != 0 || idx >= len(m.entries()) {
			out[r] = m.blankRow(specs)
			continue
		}
		out[r] = m.row(idx, specs)
	}
	if len(m.entries()) == 0 && rows > 2 {
		out[2] = m.styles.dim(fit("   "+i18n.T("Esta carpeta está vacía", "This folder is empty"), m.listW()))
	}
	return out
}

// gap is the cells between two columns: the │ separator takes one more than plain air.
func (m *Model) gap() int {
	if m.deps.Cfg.ColumnSeparators {
		return colGap + 1
	}
	return colGap
}

// gapText is what is drawn between two columns.
func (m *Model) gapText() string {
	if m.deps.Cfg.ColumnSeparators {
		return " " + m.styles.border("│") + " "
	}
	return strings.Repeat(" ", colGap)
}

// blankRow is a line of the list with no entry: only the separators go through it, so the columns read as columns
// down to the bottom of the panel.
func (m *Model) blankRow(specs []colSpec) string {
	if !m.deps.Cfg.ColumnSeparators {
		return strings.Repeat(" ", m.listW())
	}
	line := strings.Repeat(" ", m.prefix()+specs[0].w)
	for _, s := range specs[1:] {
		line += m.gapText() + strings.Repeat(" ", s.w)
	}
	return line + strings.Repeat(" ", max(0, m.listW()-ansi.StringWidth(line)))
}

func (m *Model) row(idx int, specs []colSpec) string { return m.rowW(idx, specs, m.listW()) }

// rowW draws row idx padded or cut to width cells.
func (m *Model) rowW(idx int, specs []colSpec, width int) string {
	e := m.entries()[idx]
	selected := idx == m.sel
	pointer, marker, mark := " ", " ", " "
	if m.marked[e.Path] {
		mark = "●"
	}
	switch {
	case selected:
		pointer = "▸"
	case m.isFresh(e.Path):
		pointer = "+"
	}
	if e.IsDir {
		marker = ">"
		if m.expanded[e.Path] {
			marker = "v"
		}
	}
	var b strings.Builder
	b.WriteString(pointer + mark + marker + " ")
	if m.git.ok {
		code := string(m.gitCode(e))
		if code == "" {
			code = " "
		}
		b.WriteString(m.styles.gitLetter(code) + " ")
	}
	for i, s := range specs {
		if i > 0 {
			b.WriteString(m.gapText())
		}
		text, lead := m.cell(e, s.col), ""
		if s.col == fsmodel.ColName && m.iconW() > 0 { // indent, then the icon, then the name
			indent := min(2*e.Depth, max(0, s.w-m.iconW()-1))
			ic := m.icons.Pick(icons.Item{Name: e.Name, IsDir: e.IsDir, IsLink: e.IsLink, Open: m.expanded[e.Path]})
			lead = strings.Repeat(" ", indent) + m.iconCell(ic)
			text = strings.TrimPrefix(text, strings.Repeat("  ", e.Depth))
			text = fit(text, s.w-indent-m.iconW())
		} else {
			text = fit(text, s.w)
		}
		switch {
		case s.col != fsmodel.ColName || selected:
		case m.isFresh(e.Path):
			text = m.styles.fresh(text)
		case e.IsDir:
			text = m.styles.folder(text)
		}
		b.WriteString(lead + text)
	}
	line := fit(b.String(), width)
	switch {
	case selected && m.focus == focusList:
		return m.styles.selected(ansi.Strip(line))
	case selected:
		return m.styles.selectedInactive(ansi.Strip(line))
	}
	return line
}

func (m *Model) breadcrumb() string {
	var b strings.Builder
	b.WriteString(" ")
	cs := m.crumbs()
	if x := cs[0].x; x > 1 {
		b.WriteString(m.styles.dim("…" + crumbSep))
	}
	for i, c := range cs {
		if i > 0 {
			b.WriteString(m.styles.dim(crumbSep))
		}
		if i == len(cs)-1 {
			b.WriteString(m.styles.bold(c.label))
		} else {
			b.WriteString(m.styles.title(c.label))
		}
	}
	return fit(b.String(), m.w)
}

// footerItems are the entries of the key bar, in priority order of survival: the last ones are dropped first when the
// window is narrow, except Quit, which always stays.
var footerItems = []struct {
	action keymap.Action
	label  func() string
}{
	{keymap.Open, func() string { return i18n.T("Abrir", "Open") }},
	{keymap.Preview, func() string { return i18n.T("Vista previa", "Preview") }},
	{keymap.OpenWith, func() string { return i18n.T("Abrir con", "Open with") }},
	{keymap.Search, func() string { return i18n.T("Buscar", "Search") }},
	{keymap.Sort, func() string { return i18n.T("Ordenar", "Sort") }},
	{keymap.Hidden, func() string { return i18n.T("Ocultos", "Hidden") }},
	{keymap.Focus, func() string { return i18n.T("Barra lateral", "Sidebar") }},
	{keymap.Settings, func() string { return i18n.T("Ajustes", "Settings") }},
	{keymap.Quit, func() string { return i18n.T("Salir", "Quit") }},
}

// keyLabel is how a key is written in the key bar.
func keyLabel(k string) string {
	switch k {
	case "enter":
		return "⏎"
	case "space":
		return "Space"
	case "tab":
		return "Tab"
	case "up":
		return "↑"
	case "down":
		return "↓"
	case "left":
		return "←"
	case "right":
		return "→"
	case "backspace":
		return "⌫"
	case "pgup":
		return "PgUp"
	case "pgdown":
		return "PgDn"
	}
	return k
}

func (m *Model) footerBar() string {
	if m.prompt.kind != promptNone {
		if m.status != "" && m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		return m.promptBar()
	}
	if m.goto_.open {
		if m.status != "" && m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		return m.gotoBar()
	}
	if m.searchOn() {
		if m.status != "" && m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		if m.status != "" {
			return fit(" "+m.status, m.w)
		}
		bar, _ := m.searchBar()
		return bar
	}
	if m.status != "" {
		if m.statusErr {
			return m.styles.errText(fit(" "+i18n.T("Error: ", "Error: ")+m.status, m.w))
		}
		return fit(" "+m.status, m.w)
	}
	line, _ := m.footerHints()
	return fit(line, m.w)
}

// hintSpan is a key hint of the bottom row and the action a click on it runs.
type hintSpan struct {
	x, w   int
	action keymap.Action
}

// footerHints writes the key hints that fit the window, dropping the last ones first, and says where each sits.
func (m *Model) footerHints() (string, []hintSpan) {
	build := func(skip int) (string, int, []hintSpan) {
		var parts []string
		var spans []hintSpan
		x := 1
		for i, it := range footerItems {
			if i >= len(footerItems)-skip-1 && it.action != keymap.Quit && i < len(footerItems)-1 {
				continue
			}
			ks := m.keys.Keys(it.action)
			if len(ks) == 0 {
				continue
			}
			plain := it.label() + ": " + keyLabel(ks[0])
			if len(parts) > 0 {
				x += 3 // the " | " between two hints
			}
			spans = append(spans, hintSpan{x: x, w: ansi.StringWidth(plain), action: it.action})
			x += ansi.StringWidth(plain)
			parts = append(parts, m.styles.footer(it.label()+": ")+m.styles.footerKey(keyLabel(ks[0])))
		}
		line := " " + strings.Join(parts, m.styles.dim(" | "))
		return line, ansi.StringWidth(line), spans
	}
	for skip := 0; skip < len(footerItems); skip++ {
		if line, w, spans := build(skip); w <= m.w {
			return line, spans
		}
	}
	line, _, spans := build(len(footerItems))
	return line, spans
}

// footerShowsHints says whether the bottom row holds the key hints rather than a prompt, a status or the search bar.
func (m *Model) footerShowsHints() bool {
	return m.prompt.kind == promptNone && !m.goto_.open && !m.searchOn() && m.status == ""
}

// branchLabel is the git branch with its distance from the upstream, empty outside a repository.
func (m *Model) branchLabel() string {
	if !m.git.ok || m.git.st.Branch == "" {
		return ""
	}
	l := "⎇ " + m.git.st.Branch
	if m.git.st.Ahead > 0 {
		l += fmt.Sprintf(" ↑%d", m.git.st.Ahead)
	}
	if m.git.st.Behind > 0 {
		l += fmt.Sprintf(" ↓%d", m.git.st.Behind)
	}
	return l
}

func (m *Model) minSize() [2]int {
	if m.deps.Follow {
		return [2]int{followMinWidth, followMinHeight}
	}
	return [2]int{minWidth, minHeight}
}
