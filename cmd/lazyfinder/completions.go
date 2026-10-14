package main

import (
	"fmt"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/theme"
)

// completion returns the completion script of a shell, or false when the shell is not supported.
func completion(shell string) (string, bool) {
	langs := "auto en es pt fr de it ja zh"
	themes := strings.Join(theme.Names(), " ")
	switch shell {
	case "bash":
		return fmt.Sprintf(`# bash completion for lazyfinder
_lazyfinder() {
  local cur prev
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  case "$prev" in
    --lang) COMPREPLY=($(compgen -W "%s" -- "$cur")); return ;;
    --theme) COMPREPLY=($(compgen -W "%s terminal" -- "$cur")); return ;;
    --completions|--shell-init) COMPREPLY=($(compgen -W "bash zsh fish" -- "$cur")); return ;;
    --cwd-file) COMPREPLY=($(compgen -f -- "$cur")); return ;;
    --follow) COMPREPLY=($(compgen -d -- "$cur")); return ;;
    --log) COMPREPLY=($(compgen -f -- "$cur")); return ;;
  esac
  if [[ "$cur" == -* ]]; then
    COMPREPLY=($(compgen -W "--version --print-default-config --log --log-paths --lang --theme --follow --sidebar --no-mouse --completions --cwd-file --shell-init" -- "$cur"))
  else
    COMPREPLY=($(compgen -d -- "$cur"))
  fi
}
complete -F _lazyfinder lazyfinder
`, langs, themes), true
	case "zsh":
		return fmt.Sprintf(`#compdef lazyfinder
# zsh completion for lazyfinder
_arguments \
  '--version[print the version and exit]' \
  '--print-default-config[print the default config.json and exit]' \
  '--log=[write a JSON log to a file]:file:_files' \
  '--log-paths[include file names in the log]' \
  '--lang=[interface language for this run]:language:(%s)' \
  '--theme=[theme for this run]:theme:(%s terminal)' \
  '--follow=[follow a project in a narrow tree]:directory:_files -/' \
  '--sidebar[start in the narrow project sidebar]' \
  '--no-mouse[ignore the mouse for this run]' \
  '--completions=[print a completion script]:shell:(bash zsh fish)' \
  '--cwd-file=[on exit, write the last folder to a file]:file:_files' \
  '--shell-init=[print the shell function that stays in the last folder]:shell:(bash zsh fish)' \
  '1:directory:_files -/'
`, langs, themes), true
	case "fish":
		return fmt.Sprintf(`# fish completion for lazyfinder
complete -c lazyfinder -l version -d 'Print the version and exit'
complete -c lazyfinder -l print-default-config -d 'Print the default config.json and exit'
complete -c lazyfinder -l log -r -F -d 'Write a JSON log to a file'
complete -c lazyfinder -l log-paths -d 'Include file names in the log'
complete -c lazyfinder -l lang -x -a '%s' -d 'Interface language for this run'
complete -c lazyfinder -l theme -x -a '%s terminal' -d 'Theme for this run'
complete -c lazyfinder -l follow -r -a '(__fish_complete_directories)' -d 'Follow a project in a narrow tree'
complete -c lazyfinder -l sidebar -d 'Start in the narrow project sidebar'
complete -c lazyfinder -l no-mouse -d 'Ignore the mouse for this run'
complete -c lazyfinder -l completions -x -a 'bash zsh fish' -d 'Print a completion script'
complete -c lazyfinder -l cwd-file -r -F -d 'On exit, write the last folder to a file'
complete -c lazyfinder -l shell-init -x -a 'bash zsh fish' -d 'Print the shell function that stays in the last folder'
`, langs, themes), true
	}
	return "", false
}
