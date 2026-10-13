package ui

import (
	"path/filepath"
	"slices"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// setFavorites replaces the favorites and saves them to config.json when the app has one. A path under the home
// folder is written as ~/…, the way a person writes it.
func (m *Model) setFavorites(favs []config.Place) {
	m.deps.Cfg.Sidebar.Favorites = favs
	if m.deps.ConfigPath == "" {
		return
	}
	saved := make([]config.Place, len(favs))
	for i, p := range favs {
		saved[i] = config.Place{Name: p.Name, Path: m.homeShort(p.Path)}
	}
	if err := config.Update(m.deps.ConfigPath, map[string]any{"sidebar.favorites": saved}); err != nil {
		m.setError(err)
	}
}

func (m *Model) favoriteIndex(path string) int {
	return slices.IndexFunc(m.deps.Cfg.Sidebar.Favorites, func(p config.Place) bool { return p.Path == path })
}

// toggleBookmark adds the selected folder (or, on a file or in the sidebar, the current folder) to the favorites, or
// removes it when it is already there.
func (m *Model) toggleBookmark() tea.Cmd {
	target := m.dir()
	if e, ok := m.selected(); ok && e.IsDir && m.focus != focusSidebar {
		target = e.Path
	}
	if m.favoriteIndex(target) >= 0 {
		return m.removeFavorite(target)
	}
	return m.addFavorite(target)
}

func (m *Model) addFavorite(path string) tea.Cmd {
	if m.favoriteIndex(path) >= 0 {
		return nil
	}
	name := filepath.Base(path)
	if path == "/" {
		name = "/"
	}
	m.setFavorites(append(slices.Clone(m.deps.Cfg.Sidebar.Favorites), config.Place{Name: name, Path: path}))
	m.setStatus(i18n.T("Añadido a Favoritos", "Bookmarked"))
	return nil
}

func (m *Model) removeFavorite(path string) tea.Cmd {
	i := m.favoriteIndex(path)
	if i < 0 {
		return nil
	}
	m.setFavorites(slices.Delete(slices.Clone(m.deps.Cfg.Sidebar.Favorites), i, i+1))
	m.setStatus(i18n.T("Marcador quitado", "Bookmark removed"))
	if dests := destinations(m.places()); m.sideSel >= len(dests) {
		m.sideSel = max(0, len(dests)-1)
	}
	return nil
}

func (m *Model) renameFavorite(path, name string) tea.Cmd {
	i := m.favoriteIndex(path)
	if i < 0 || name == "" {
		return nil
	}
	favs := slices.Clone(m.deps.Cfg.Sidebar.Favorites)
	favs[i].Name = name
	m.setFavorites(favs)
	return nil
}

// moveFavorite moves a favorite up (-1) or down (+1) and keeps the sidebar selection on it.
func (m *Model) moveFavorite(path string, delta int) {
	i := m.favoriteIndex(path)
	j := i + delta
	if i < 0 || j < 0 || j >= len(m.deps.Cfg.Sidebar.Favorites) {
		return
	}
	favs := slices.Clone(m.deps.Cfg.Sidebar.Favorites)
	favs[i], favs[j] = favs[j], favs[i]
	m.setFavorites(favs)
	m.sideSel += delta
}

// selectedPlace is the place the sidebar selection is on.
func (m *Model) selectedPlace() (place, bool) {
	ps := m.places()
	dests := destinations(ps)
	if len(dests) == 0 {
		return place{}, false
	}
	return ps[dests[min(m.sideSel, len(dests)-1)]], true
}

// favoriteKey handles the keys that edit the favorites while the sidebar has the focus.
func (m *Model) favoriteKey(key string, act keymap.Action) (tea.Cmd, bool) {
	p, ok := m.selectedPlace()
	switch {
	case act == keymap.Bookmark:
		return m.toggleBookmark(), true
	case !ok || p.fav == 0:
		return nil, false
	case act == keymap.Trash:
		return m.removeFavorite(p.path), true
	case act == keymap.Rename:
		m.prompt = promptState{kind: promptPlace, input: m.deps.Cfg.Sidebar.Favorites[p.fav-1].Name, target: p.path}
		return nil, true
	case key == "K" || key == "alt+up":
		m.moveFavorite(p.path, -1)
		return nil, true
	case key == "J" || key == "alt+down":
		m.moveFavorite(p.path, 1)
		return nil, true
	}
	return nil, false
}

// ctxItem is one line of a context menu.
type ctxItem struct {
	label string
	run   func() tea.Cmd
}

// ctxState is the small menu a right click opens.
type ctxState struct {
	open  bool
	x, y  int
	title string
	items []ctxItem
	sel   int
}

// openContext builds the right-click menu of what lies under the pointer; it returns false when there is none.
func (m *Model) openContext(z zone, x, y int) bool {
	var items []ctxItem
	title := ""
	switch z.kind {
	case zonePlace:
		p := m.places()[destinations(m.places())[z.idx]]
		if p.recents {
			return false
		}
		title = p.name
		items = append(items, ctxItem{i18n.T("Abrir", "Open"), func() tea.Cmd { return m.goPlace(p) }})
		if p.fav > 0 {
			items = append(items,
				ctxItem{i18n.T("Renombrar", "Rename"), func() tea.Cmd {
					m.prompt = promptState{kind: promptPlace, input: m.deps.Cfg.Sidebar.Favorites[p.fav-1].Name, target: p.path}
					return nil
				}},
				ctxItem{i18n.T("Subir", "Move up"), func() tea.Cmd { m.moveFavorite(p.path, -1); return nil }},
				ctxItem{i18n.T("Bajar", "Move down"), func() tea.Cmd { m.moveFavorite(p.path, 1); return nil }},
				ctxItem{i18n.T("Quitar de Favoritos", "Remove from Favorites"), func() tea.Cmd { return m.removeFavorite(p.path) }})
		}
	case zoneRow:
		e := m.entries()[z.idx]
		if !e.IsDir {
			return false
		}
		title = e.Name
		if m.favoriteIndex(e.Path) >= 0 {
			items = append(items, ctxItem{i18n.T("Quitar de Favoritos", "Remove from Favorites"), func() tea.Cmd { return m.removeFavorite(e.Path) }})
		} else {
			items = append(items, ctxItem{i18n.T("Añadir a Favoritos", "Add to Favorites"), func() tea.Cmd { return m.addFavorite(e.Path) }})
		}
	case zoneHeader, zoneColBorder:
		items = m.columnMenu(z.col)
		title = i18n.T("Columnas", "Columns")
	default:
		return false
	}
	m.ctx = ctxState{open: true, x: x, y: y, title: title, items: items}
	return true
}

func (m *Model) ctxInner() int {
	w := ansi.StringWidth(m.ctx.title) + 4
	for _, it := range m.ctx.items {
		w = max(w, ansi.StringWidth(it.label)+4)
	}
	return min(w, m.w-6)
}

func (m *Model) ctxBox() (x, y int, lines []string) {
	iw := m.ctxInner()
	content := make([]string, len(m.ctx.items))
	for i, it := range m.ctx.items {
		line := "  " + it.label
		if i == m.ctx.sel {
			line = m.styles.selected(ansi.Strip(fit(line, iw)))
		}
		content[i] = line
	}
	x = max(0, min(m.ctx.x, m.w-iw-4))
	y = max(0, min(m.ctx.y, m.h-len(content)-2))
	return x, y, m.box(m.ctx.title, content, iw)
}

func (m *Model) ctxKey(key string, act keymap.Action) tea.Cmd {
	n := len(m.ctx.items)
	switch {
	case act == keymap.Quit:
		return tea.Quit
	case key == "esc":
		m.ctx = ctxState{}
	case act == keymap.Up:
		m.ctx.sel = max(0, m.ctx.sel-1)
	case act == keymap.Down:
		m.ctx.sel = min(n-1, m.ctx.sel+1)
	case key == "enter":
		return m.ctxRun(m.ctx.sel)
	}
	return nil
}

func (m *Model) ctxRun(i int) tea.Cmd {
	run := m.ctx.items[i].run
	m.ctx = ctxState{}
	return run()
}

func (m *Model) ctxClick(x, y int) tea.Cmd {
	mx, my, bx := m.ctxBox()
	if !inBox(x, y, mx, my, bx) {
		m.ctx = ctxState{}
		return nil
	}
	if i := y - my - 1; i >= 0 && i < len(m.ctx.items) {
		return m.ctxRun(i)
	}
	return nil
}
