// Package opener decides which program opens a file and starts it without a shell
// (docs/adr/0005-open-with-safety.md).
package opener

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
)

// DefaultName names the built-in system opener.
const DefaultName = "default"

// Command is a resolved invocation: argv[0] is the program, the rest are arguments.
type Command struct {
	Name string
	Mode string
	Argv []string
	Dir  string // the folder to run in; empty means the current one
}

// Resolver maps files to commands using the user's configuration.
type Resolver struct{ cfg config.Config }

// New builds a Resolver.
func New(cfg config.Config) Resolver { return Resolver{cfg: cfg} }

// Resolve picks the command for path: the opener configured for its extension, else the system default.
func (r Resolver) Resolve(path string) Command {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	if name, ok := r.cfg.OpenWith[ext]; ok {
		return r.named(name, path)
	}
	return systemDefault(path)
}

// Choices lists every opener that can be chosen for path: the configured ones in name order, then the default.
func (r Resolver) Choices(path string) []Command {
	var out []Command
	for _, n := range r.cfg.OpenerNames() {
		out = append(out, r.named(n, path))
	}
	for _, n := range r.cfg.BuiltinNames() {
		if c := r.named(n, path); appInstalled(c.Argv) {
			out = append(out, c)
		}
	}
	return append(out, systemDefault(path))
}

func (r Resolver) named(name, path string) Command {
	op, _ := r.cfg.Opener(name)
	argv := make([]string, len(op.Argv))
	for i, a := range op.Argv {
		if a == config.FilePlaceholder {
			argv[i] = safeArg(path)
			continue
		}
		argv[i] = a
	}
	return Command{Name: name, Mode: op.Mode, Argv: argv}
}

func systemDefault(path string) Command {
	return Command{Name: DefaultName, Mode: config.ModeGUI, Argv: []string{"open", "--", path}}
}

// safeArg keeps a leading dash from being read as an option.
func safeArg(path string) string {
	if strings.HasPrefix(path, "-") {
		return "./" + path
	}
	return path
}

// Custom builds a user command: {file} and {dir} are replaced as whole arguments and the command runs in dir.
func Custom(name string, c config.Opener, file, dir string) Command {
	argv := make([]string, len(c.Argv))
	for i, a := range c.Argv {
		switch a {
		case config.FilePlaceholder:
			argv[i] = safeArg(file)
		case config.DirPlaceholder:
			argv[i] = dir
		default:
			argv[i] = a
		}
	}
	return Command{Name: name, Mode: c.Mode, Argv: argv, Dir: dir}
}

// DirRunner is a Runner that can also start a process in a given folder.
type DirRunner interface {
	Runner
	StartIn(dir string, argv []string) error
}

// Runner starts a process. Tests inject a recorder.
type Runner interface {
	Start(argv []string) error
}

// ExecRunner starts processes with os/exec, never through a shell, and does not wait for them.
type ExecRunner struct{}

// Start launches argv detached from lazyfinder's lifetime.
func (r ExecRunner) Start(argv []string) error { return r.StartIn("", argv) }

// StartIn is Start with the working folder set (empty keeps the current one).
func (ExecRunner) StartIn(dir string, argv []string) error {
	if len(argv) == 0 {
		return errors.New("empty command")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cannot start %q: %w", argv[0], err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// Open starts a GUI command. Terminal-mode commands need the UI to hand the terminal over, so Open refuses them.
func (Resolver) Open(run Runner, c Command) error {
	if c.Mode == config.ModeTerminal {
		return fmt.Errorf("opener %q runs in the terminal: it must be started by the UI", c.Name)
	}
	if dr, ok := run.(DirRunner); ok && c.Dir != "" {
		return dr.StartIn(c.Dir, c.Argv)
	}
	return run.Start(c.Argv)
}

// QuickLook is the documented fallback for previews the terminal cannot draw: macOS Quick Look in its own window,
// started only when the user presses Q (docs/adr/0006-preview-and-images.md).
func QuickLook(path string) Command {
	return Command{Name: "quicklook", Mode: config.ModeGUI, Argv: []string{"qlmanage", "-p", safeArg(path)}}
}

// AppExists says whether a macOS application bundle is installed. It is a variable so tests do not depend on the
// applications of the machine they run on.
var AppExists = func(name string) bool {
	home, _ := os.UserHomeDir()
	for _, dir := range []string{"/Applications", "/System/Applications", "/System/Applications/Utilities", filepath.Join(home, "Applications")} {
		if _, err := os.Stat(filepath.Join(dir, name+".app")); err == nil {
			return true
		}
	}
	return false
}

// appInstalled is false for an "open -a <App>" command whose application is missing; other commands pass.
func appInstalled(argv []string) bool {
	if len(argv) >= 3 && argv[0] == "open" && argv[1] == "-a" {
		return AppExists(argv[2])
	}
	return true
}
