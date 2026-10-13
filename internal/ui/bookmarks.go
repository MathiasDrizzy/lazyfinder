package ui

import (
	"path/filepath"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// toggleBookmark adds the selected folder (or, on a file, the current folder) to the sidebar favorites, or removes
// it when it is already there. The change is saved to config.json when the app has one.
func (m *Model) toggleBookmark() tea.Cmd {
	target := m.dir()
	if e, ok := m.selected(); ok && e.IsDir {
		target = e.Path
	}
	favs := slices.Clone(m.deps.Cfg.Sidebar.Favorites)
	if i := slices.IndexFunc(favs, func(p config.Place) bool { return p.Path == target }); i >= 0 {
		favs = slices.Delete(favs, i, i+1)
		m.setStatus(i18n.T("Marcador quitado", "Bookmark removed"))
	} else {
		name := filepath.Base(target)
		if target == "/" {
			name = "/"
		}
		favs = append(favs, config.Place{Name: name, Path: target})
		m.setStatus(i18n.T("Añadido a Favoritos", "Bookmarked"))
	}
	m.deps.Cfg.Sidebar.Favorites = favs
	if m.deps.ConfigPath != "" {
		saved := make([]config.Place, len(favs))
		for i, p := range favs {
			saved[i] = config.Place{Name: p.Name, Path: m.homeShort(p.Path)}
		}
		if err := config.Update(m.deps.ConfigPath, map[string]any{"sidebar.favorites": saved}); err != nil {
			m.setError(err)
		}
	}
	return nil
}

// homeShort writes a path under the home folder as ~/…, the way a person writes it in config.json.
func (m *Model) homeShort(p string) string {
	home := m.deps.Home
	switch {
	case home == "":
		return p
	case p == home:
		return "~"
	case strings.HasPrefix(p, home+string(filepath.Separator)):
		return "~/" + filepath.ToSlash(p[len(home)+1:])
	}
	return p
}
