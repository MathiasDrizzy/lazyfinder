package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// Git operations run off the UI thread: a fetch can take a minute and the window must keep answering. The result
// comes back as a gitDoneMsg, which says what happened on the bottom row and reads the repository again.
type gitDoneMsg struct {
	done string
	err  error
}

// gitRun starts op on the repository root. busy is shown while it runs and done when it worked; an error is shown as
// git said it (explainGit), because that is the only text that names the file or the branch that is in the way.
func (m *Model) gitRun(busy, done string, op func(root string) error) tea.Cmd {
	if !m.git.ok {
		m.setStatus(i18n.T("No es un repositorio git", "Not a git repository"))
		return nil
	}
	root := m.git.st.Root
	m.setStatus(busy)
	return func() tea.Msg { return gitDoneMsg{done: done, err: op(root)} }
}

func (m *Model) onGitDone(msg gitDoneMsg) tea.Cmd {
	if msg.err != nil {
		m.setError(errors.New(explainGit(msg.err.Error())))
	} else {
		m.setStatus(msg.done)
	}
	return m.refresh("")
}

// confirm asks a yes/no question on the bottom row and runs yes on "y".
func (m *Model) confirm(ask string, yes func() tea.Cmd) {
	m.prompt = promptState{kind: promptConfirm, ask: ask, yes: yes}
}

// lazygitPath is where lazygit is, empty when it is not installed.
func (m *Model) lazygitPath() string {
	look := m.deps.LookPath
	if look == nil {
		look = exec.LookPath
	}
	p, err := look("lazygit")
	if err != nil {
		return ""
	}
	return p
}

// openLazygit hands the terminal to lazygit in the repository: committing, rebasing and merging live there.
func (m *Model) openLazygit() tea.Cmd {
	p := m.lazygitPath()
	if p == "" {
		m.setStatus(i18n.T("lazygit no está instalado", "lazygit is not installed"))
		return nil
	}
	if !m.git.ok {
		m.setStatus(i18n.T("No es un repositorio git", "Not a git repository"))
		return nil
	}
	cmd := exec.Command(p)
	cmd.Dir = m.git.st.Root
	return m.handOver(cmd)
}

// explainGit is what git said as one short sentence for the bottom row. The refusal to overwrite changes is the one
// worth rewriting: git spends three lines on it and a narrow window cuts off the file that is in the way, so the
// sentence names the file first. Any other message is kept as git said it, without its error: or fatal: prefix, which
// the row already says.
func explainGit(msg string) string {
	parts := strings.Split(msg, " · ")
	head := strings.TrimSpace(parts[0])
	if !strings.Contains(head, "would be overwritten by") {
		return stripGitPrefix(msg)
	}
	var files []string
	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)
		if p == "" || strings.HasPrefix(p, "Please ") || strings.HasPrefix(p, "Aborting") {
			continue
		}
		files = append(files, filepath.Base(p))
	}
	if len(files) == 0 {
		return stripGitPrefix(msg)
	}
	what := files[0]
	if n := len(files) - 1; n > 0 {
		what += fmt.Sprintf(i18n.T(" y %d más", " and %d more"), n)
	}
	if strings.HasSuffix(head, "by merge:") {
		return fmt.Sprintf(i18n.T("Cambios sin guardar en %s: haz commit o stash antes de fusionar",
			"Unsaved changes in %s: commit or stash before merging"), what)
	}
	return fmt.Sprintf(i18n.T("Cambios sin guardar en %s: haz commit o stash antes de cambiar de rama",
		"Unsaved changes in %s: commit or stash before switching branches"), what)
}

// stripGitPrefix drops the error: or fatal: that begins what git said.
func stripGitPrefix(msg string) string {
	for _, p := range []string{"error: ", "fatal: "} {
		if rest, ok := strings.CutPrefix(msg, p); ok {
			return rest
		}
	}
	return msg
}
