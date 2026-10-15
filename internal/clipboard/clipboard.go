// Package clipboard copies text with the clipboard tool of the system: pbcopy on macOS, wl-copy on Wayland, xclip or
// xsel on X11. When none is installed (over SSH, for example) the caller falls back to OSC 52, which asks the
// terminal itself to set the clipboard.
package clipboard

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNoTool means no clipboard tool is installed.
var ErrNoTool = errors.New("no clipboard tool installed")

const timeout = 2 * time.Second

// tools are tried in order; the first one on PATH takes the text.
var tools = [][]string{
	{"pbcopy"},
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
}

// Copy puts text on the clipboard with the first tool found on PATH.
func Copy(text string) error {
	for _, argv := range tools {
		path, err := exec.LookPath(argv[0])
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, argv[1:]...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%s: %w", argv[0], err)
		}
		return nil
	}
	return ErrNoTool
}
