package main

import (
	"path/filepath"

	"github.com/MathiasDrizzy/lazyfinder/internal/ui"
)

// shellInit returns the function that wraps lazyfinder in a shell: it runs lazyfinder with --cwd-file and, when
// lazyfinder wrote a folder there, moves the shell to it (the way of yazi's y and lf's lfcd). It goes in the shell's
// startup file: eval "$(lazyfinder --shell-init zsh)".
func shellInit(shell string) (string, bool) {
	switch shell {
	case "bash", "zsh":
		return `# lazyfinder: leave the shell in the folder lazyfinder ends in (cd_on_exit in Settings)
lazyfinder() {
  local cwd_file cwd code
  cwd_file="$(mktemp -t lazyfinder-cwd.XXXXXX)" || return
  command lazyfinder --cwd-file "$cwd_file" "$@"
  code=$?
  cwd="$(command cat -- "$cwd_file")"
  if [ -n "$cwd" ] && [ "$cwd" != "$PWD" ] && [ -d "$cwd" ]; then
    builtin cd -- "$cwd" || code=$?
  fi
  command rm -f -- "$cwd_file"
  return $code
}
`, true
	case "fish":
		return `# lazyfinder: leave the shell in the folder lazyfinder ends in (cd_on_exit in Settings)
function lazyfinder
    set -l cwd_file (mktemp -t lazyfinder-cwd.XXXXXX); or return
    command lazyfinder --cwd-file $cwd_file $argv
    set -l code $status
    set -l cwd (command cat -- $cwd_file)
    if test -n "$cwd"; and test "$cwd" != "$PWD"; and test -d "$cwd"
        builtin cd -- $cwd
    end
    command rm -f -- $cwd_file
    return $code
end
`, true
	}
	return "", false
}

// shellSetup is the line that installs the function of --shell-init for the shell at path ($SHELL), and the file it
// goes in. A shell without a function (tcsh, nu) gets the zsh line, the macOS default.
func shellSetup(path string) ui.ShellSetup {
	switch filepath.Base(path) {
	case "bash":
		return ui.ShellSetup{File: "~/.bashrc", Line: `eval "$(lazyfinder --shell-init bash)"`}
	case "fish":
		return ui.ShellSetup{File: "~/.config/fish/config.fish", Line: "lazyfinder --shell-init fish | source"}
	}
	return ui.ShellSetup{File: "~/.zshrc", Line: `eval "$(lazyfinder --shell-init zsh)"`}
}
