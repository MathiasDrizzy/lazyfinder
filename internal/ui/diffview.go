package ui

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/keymap"
	"github.com/charmbracelet/x/ansi"
)

// diffState is the popup that shows what changed in one file.
type diffState struct {
	open   bool
	title  string
	lines  []string
	scroll int
}

// openDiff shows the patch of one change. A staged change shows the index against the last commit.
func (m *Model) openDiff(c git.Change, staged bool) tea.Cmd {
	if !m.git.ok {
		return nil
	}
	out, err := git.Diff(m.git.st.Root, c, staged)
	if err != nil {
		m.setError(err)
		return nil
	}
	if strings.TrimSpace(out) == "" {
		out = i18n.T("Sin diferencias de texto (¿binario, vacío o solo permisos?)", "No text differences (binary, empty or only permissions?)")
	}
	title := c.Path
	if staged {
		title += " · " + i18n.T("preparado", "staged")
	}
	m.diff = diffState{open: true, title: title, lines: strings.Split(strings.TrimRight(out, "\n"), "\n")}
	return nil
}

func (m *Model) diffBody() int {
	_, _, _, _, _, bodyH := m.previewGeom()
	return max(1, bodyH)
}

func (m *Model) diffMax() int { return max(0, len(m.diff.lines)-m.diffBody()) }

// diffLine colors one line of a patch: green added, red removed, accent for the @@ headers, dim for the file header.
func (m *Model) diffLine(l string) string {
	l = strings.ReplaceAll(l, "\t", "    ")
	switch {
	case strings.HasPrefix(l, "+++"), strings.HasPrefix(l, "---"), strings.HasPrefix(l, "diff "), strings.HasPrefix(l, "index "):
		return m.styles.dim(l)
	case strings.HasPrefix(l, "@@"):
		return m.styles.accent(l)
	case strings.HasPrefix(l, "+"):
		return m.styles.gitPaint("A", l)
	case strings.HasPrefix(l, "-"):
		return m.styles.gitPaint("D", l)
	}
	return l
}

func (m *Model) diffBox() (int, int, []string) {
	x, y, _, _, iw, bodyH := m.previewGeom()
	body := make([]string, 0, bodyH)
	for i := m.diff.scroll; i < len(m.diff.lines) && len(body) < bodyH; i++ {
		body = append(body, m.diffLine(ansi.Truncate(m.diff.lines[i], iw, "…")))
	}
	for len(body) < bodyH {
		body = append(body, "")
	}
	pos := ""
	if m.diffMax() > 0 {
		pos = " " + strconv.Itoa(m.diff.scroll+1) + "/" + strconv.Itoa(len(m.diff.lines))
	}
	body = append(body, "", m.styles.dim(fit(i18n.T("↑↓ desplazar · Esc cerrar", "↑↓ scroll · Esc close")+pos, iw)))
	return x, y, m.box(m.diff.title, body, iw)
}

func (m *Model) diffKey(key string, act keymap.Action) tea.Cmd {
	page := m.diffBody()
	switch {
	case key == "esc" || act == keymap.Quit:
		m.diff = diffState{}
	case act == keymap.Up:
		m.diff.scroll = max(0, m.diff.scroll-1)
	case act == keymap.Down:
		m.diff.scroll = min(m.diffMax(), m.diff.scroll+1)
	case act == keymap.PageUp:
		m.diff.scroll = max(0, m.diff.scroll-page)
	case act == keymap.PageDown, key == "space":
		m.diff.scroll = min(m.diffMax(), m.diff.scroll+page)
	case act == keymap.First:
		m.diff.scroll = 0
	case act == keymap.Last:
		m.diff.scroll = m.diffMax()
	}
	return nil
}
