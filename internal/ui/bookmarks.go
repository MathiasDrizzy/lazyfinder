package ui

import (
	"path/filepath"
	"strings"
)

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
