package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/git"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/charmbracelet/x/ansi"
)

// gmItem is one line of the branch menu.
type gmItem struct {
	label, hint string
	current     bool
	run         func() tea.Cmd
}

// gitMenuState is the popup of the branches, like herdr's branch picker: type to search, Enter switches, a name that
// does not exist offers to create it, and the repository-wide actions (fetch, pull, push, stash) sit below.
type gitMenuState struct {
	open     bool
	query    string
	sel      int
	branches []git.Branch
	stashes  int
}

// openGitMenu opens the branch menu on the repository of the folder being shown.
func (m *Model) openGitMenu() tea.Cmd {
	if !m.git.ok {
		m.setStatus(i18n.T("No es un repositorio git", "Not a git repository"))
		return nil
	}
	bs, err := git.Branches(m.git.st.Root)
	if err != nil {
		m.setError(err)
		return nil
	}
	list, _ := git.Stashes(m.git.st.Root)
	m.gm = gitMenuState{open: true, branches: bs, stashes: len(list)}
	for i, b := range bs {
		if b.Current {
			m.gm.sel = i
		}
	}
	return nil
}

func contains(s, q string) bool { return strings.Contains(strings.ToLower(s), strings.ToLower(q)) }

// gmItems are the lines for the current query: the matching branches, a line to create the name when no branch has
// it, then the matching actions. Enter on a half-typed name switches to the branch it finds, never creates by surprise.
func (m *Model) gmItems() []gmItem {
	q := strings.TrimSpace(m.gm.query)
	var items []gmItem
	exists := false
	for _, b := range m.gm.branches {
		if b.Name == q {
			exists = true
		}
	}
	for _, b := range m.gm.branches {
		if q != "" && !contains(b.Name, q) {
			continue
		}
		b := b
		it := gmItem{label: b.Name, current: b.Current, run: func() tea.Cmd {
			if b.Current {
				return nil
			}
			return m.gitRun(i18n.T("Cambiando de rama…", "Switching branch…"), fmt.Sprintf(i18n.T("En la rama %s", "On branch %s"), b.Name),
				func(root string) error { return git.Checkout(root, b.Name) })
		}}
		switch {
		case b.Current:
			it.hint = i18n.T("actual", "current")
		case b.Remote != "":
			it.hint = b.Remote
		}
		items = append(items, it)
	}
	if q != "" && !exists && git.ValidBranch(m.git.st.Root, q) == nil {
		name := q
		items = append(items, gmItem{
			label: fmt.Sprintf(i18n.T("+ Crear rama «%s»", "+ Create branch “%s”"), name),
			run: func() tea.Cmd {
				return m.gitRun(i18n.T("Creando la rama…", "Creating the branch…"), fmt.Sprintf(i18n.T("Rama %s creada", "Created branch %s"), name),
					func(root string) error { return git.CreateBranch(root, name) })
			},
		})
	}
	for _, a := range m.gmActions() {
		if q == "" || contains(a.label, q) {
			items = append(items, a)
		}
	}
	return items
}

// gmActions are the repository-wide actions of the menu.
func (m *Model) gmActions() []gmItem {
	acts := []gmItem{
		{label: i18n.T("Traer cambios (fetch)", "Fetch"), run: m.gitFetch},
		{label: i18n.T("Bajar (pull, solo avance rápido)", "Pull (fast-forward only)"), run: m.gitPull},
		{label: i18n.T("Subir (push)", "Push"), run: m.gitPush},
	}
	if m.git.st.Changed() > 0 {
		acts = append(acts, gmItem{label: i18n.T("Guardar cambios en el stash", "Stash changes"), run: m.gitStash})
	}
	if m.gm.stashes > 0 {
		acts = append(acts, gmItem{label: i18n.T("Recuperar el último stash", "Pop the latest stash"), run: m.gitStashPop})
	}
	if m.lazygitPath() != "" {
		acts = append(acts, gmItem{label: i18n.T("Abrir lazygit (commit, rebase, merge)", "Open lazygit (commit, rebase, merge)"), run: m.openLazygit})
	}
	return acts
}

func (m *Model) gitFetch() tea.Cmd {
	return m.gitRun(i18n.T("Trayendo cambios…", "Fetching…"), i18n.T("Al día con el remoto", "Fetched"), git.Fetch)
}

func (m *Model) gitPull() tea.Cmd {
	return m.gitRun(i18n.T("Bajando…", "Pulling…"), i18n.T("Rama actualizada", "Pulled"), git.Pull)
}

func (m *Model) gitPush() tea.Cmd {
	return m.gitRun(i18n.T("Subiendo…", "Pushing…"), i18n.T("Rama subida", "Pushed"), git.Push)
}

func (m *Model) gitStash() tea.Cmd {
	return m.gitRun(i18n.T("Guardando en el stash…", "Stashing…"), i18n.T("Cambios guardados en el stash", "Changes stashed"),
		func(root string) error { return git.StashPush(root, "") })
}

func (m *Model) gitStashPop() tea.Cmd {
	return m.gitRun(i18n.T("Recuperando el stash…", "Popping the stash…"), i18n.T("Stash recuperado", "Stash restored"), git.StashPop)
}

// gmRows is how many items fit.
func (m *Model) gmRows() int { return max(1, min(len(m.gmItems()), m.h-12)) }

func (m *Model) gmTop(rows, n int) int { return max(0, min(m.gm.sel-rows+1, n-rows)) }

func (m *Model) gitMenuGeom() (x, y, iw int) {
	iw = max(14, min(46, m.w-8))
	return (m.w - iw - 4) / 2, 2, iw
}

// dirtyWarning says how many changes go along when the branch changes.
func (m *Model) dirtyWarning() string {
	switch n := m.git.st.Changed(); n {
	case 0:
		return ""
	case 1:
		return i18n.T("⚠ 1 cambio sin guardar viaja contigo", "⚠ 1 uncommitted change goes with you")
	default:
		return fmt.Sprintf(i18n.T("⚠ %d cambios sin guardar viajan contigo", "⚠ %d uncommitted changes go with you"), n)
	}
}

func (m *Model) gitMenuBox() (int, int, []string) {
	x, y, iw := m.gitMenuGeom()
	items := m.gmItems()
	rows := m.gmRows()
	sel := min(m.gm.sel, max(0, len(items)-1))
	top := max(0, min(sel-rows+1, len(items)-rows))
	content := []string{m.styles.accent(fit(" ⎇ "+m.gm.query+"▏", iw))}
	if w := m.dirtyWarning(); w != "" {
		content = append(content, m.styles.dim(fit(w, iw)))
	} else {
		content = append(content, "")
	}
	for i := top; i < top+rows && i < len(items); i++ {
		it := items[i]
		pointer := "  "
		if it.current {
			pointer = "● "
		}
		if i == sel {
			pointer = "▸ "
		}
		hintW := min(ansi.StringWidth(it.hint), iw/2)
		line := fit(pointer+it.label, iw-hintW-1) + " " + fit(it.hint, hintW)
		if i == sel {
			line = m.styles.selected(ansi.Strip(line))
		} else if it.hint != "" {
			line = fit(pointer+it.label, iw-hintW-1) + " " + m.styles.dim(fit(it.hint, hintW))
		}
		content = append(content, line)
	}
	if len(items) == 0 {
		content = append(content, m.styles.dim(fit("  "+i18n.T("Nada coincide", "Nothing matches"), iw)))
	}
	return x, y, m.box(i18n.T("Ramas", "Branches"), content, iw)
}

func (m *Model) gitMenuKey(k tea.KeyPressMsg) tea.Cmd {
	items := m.gmItems()
	n := len(items)
	switch key := k.String(); key {
	case "esc":
		m.gm = gitMenuState{}
	case "up":
		m.gm.sel = wrap(m.gm.sel-1, n)
	case "down":
		m.gm.sel = wrap(m.gm.sel+1, n)
	case "home":
		m.gm.sel = 0
	case "end":
		m.gm.sel = max(0, n-1)
	case "pgup":
		m.gm.sel = max(0, m.gm.sel-m.gmRows())
	case "pgdown":
		m.gm.sel = min(max(0, n-1), m.gm.sel+m.gmRows())
	case "enter":
		return m.gmChoose(m.gm.sel)
	case "backspace":
		if r := []rune(m.gm.query); len(r) > 0 {
			m.gm.query, m.gm.sel = string(r[:len(r)-1]), 0
		}
	case "ctrl+u":
		m.gm.query, m.gm.sel = "", 0
	default:
		if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			m.gm.query += k.Text
			m.gm.sel = 0
		}
	}
	return nil
}

func (m *Model) gmChoose(i int) tea.Cmd {
	items := m.gmItems()
	if i < 0 || i >= len(items) {
		return nil
	}
	run := items[i].run
	m.gm = gitMenuState{}
	return run()
}

func (m *Model) gitMenuClick(x, y int) tea.Cmd {
	px, py, bx := m.gitMenuBox()
	if !inBox(x, y, px, py, bx) {
		return nil
	}
	rows, items := m.gmRows(), m.gmItems()
	sel := min(m.gm.sel, max(0, len(items)-1))
	top := max(0, min(sel-rows+1, len(items)-rows))
	if r := y - py - 3; r >= 0 && r < rows && top+r < len(items) { // the frame, the query and the warning come first
		return m.gmChoose(top + r)
	}
	return nil
}
