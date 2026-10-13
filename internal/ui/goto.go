package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// gotoState is the "Go to folder" prompt on the bottom row.
type gotoState struct {
	open  bool
	input string
}

func (m *Model) gotoKey(k tea.KeyPressMsg) tea.Cmd {
	key := k.String()
	switch key {
	case "esc":
		m.goto_ = gotoState{}
	case "enter":
		return m.gotoSubmit()
	case "tab":
		m.goto_.input = completeFolder(m.goto_.input, m.dir(), m.deps.Home)
	case "backspace":
		if r := []rune(m.goto_.input); len(r) > 0 {
			m.goto_.input = string(r[:len(r)-1])
		}
	case "ctrl+u":
		m.goto_.input = ""
	default:
		if k.Text != "" && k.Mod&(tea.ModCtrl|tea.ModAlt) == 0 {
			m.goto_.input += k.Text
		}
	}
	return nil
}

func (m *Model) gotoSubmit() tea.Cmd {
	target := resolvePath(m.goto_.input, m.dir(), m.deps.Home)
	info, err := os.Stat(target)
	if err != nil {
		m.setError(fmt.Errorf(i18n.T("%s no existe", "%s does not exist"), target))
		return nil
	}
	name := ""
	if !info.IsDir() {
		target, name = filepath.Dir(target), filepath.Base(target)
	}
	m.goto_ = gotoState{}
	return m.enter(target, name)
}

// resolvePath turns what the user typed into an absolute path: ~ is the home, a relative path starts at dir.
func resolvePath(in, dir, home string) string {
	in = strings.TrimSpace(in)
	switch {
	case in == "~":
		return home
	case strings.HasPrefix(in, "~/"):
		return filepath.Join(home, in[2:])
	case filepath.IsAbs(in):
		return filepath.Clean(in)
	}
	return filepath.Join(dir, in)
}

// completeFolder completes the last path element of in with the folders that start with it: one match is
// completed with a trailing slash, several with their common prefix.
func completeFolder(in, dir, home string) string {
	slash := strings.LastIndex(in, "/")
	head, prefix := in[:slash+1], in[slash+1:]
	parent := resolvePath(head, dir, home)
	if head == "" {
		parent = dir
	}
	des, err := os.ReadDir(parent)
	if err != nil {
		return in
	}
	var names []string
	for _, de := range des {
		n := de.Name()
		if !strings.HasPrefix(n, prefix) || (strings.HasPrefix(n, ".") && !strings.HasPrefix(prefix, ".")) {
			continue
		}
		if de.IsDir() || isDirLink(filepath.Join(parent, n)) {
			names = append(names, n)
		}
	}
	switch len(names) {
	case 0:
		return in
	case 1:
		return head + names[0] + "/"
	}
	sort.Strings(names)
	return head + commonPrefix(names)
}

func isDirLink(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func commonPrefix(names []string) string {
	p := names[0]
	for _, n := range names[1:] {
		for !strings.HasPrefix(n, p) {
			p = p[:len(p)-1]
		}
	}
	return p
}

func (m *Model) gotoBar() string {
	return fit(m.styles.accent(" "+i18n.T("Ir a: ", "Go to: "))+m.goto_.input+"▏", m.w)
}
