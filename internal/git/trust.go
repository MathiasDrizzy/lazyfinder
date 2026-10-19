package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// commandKeys are the settings that make git run a program while it only reads the working tree: the fsmonitor hook
// on status and the clean, smudge and process filters on any file git has to compare. textconv is turned off with
// --no-textconv where a diff is shown, and external diff drivers with --no-ext-diff.
const commandKeys = `^(core\.fsmonitor|filter\..+\.(clean|smudge|process))$`

// readOnly returns the options and the environment that switch off every commandKeys setting the repository itself
// defines, in .git/config, a file it includes or a worktree config. A folder you only look at may come from anywhere
// (a zip, a shared drive), and looking must not run what its config says. The same settings from your global or system
// config, or from the command line, are yours and stay on, so git-lfs and an fsmonitor daemon set up there keep
// working. Asking git config runs nothing. A setting goes as -c key=, which every git reads, unless its name has an =:
// -c splits at the first one, so -c filter.x=y.clean= set filter.x and left a filter named "x=y" on. Those go as
// GIT_CONFIG_KEY_n and GIT_CONFIG_VALUE_n, after any the environment already has, and a git older than 2.31, which
// ignores them, does not read such a repository at all. A submodule has a config of its own that this does not read,
// so the reads pass --ignore-submodules=dirty: git then compares the submodule's commit and never runs a status in it.
func readOnly(ctx context.Context, dir string) (args, env []string, err error) {
	out, err := git(ctx, dir, "config", "--includes", "--show-scope", "--name-only", "--get-regexp", commandKeys)
	if err != nil {
		return nil, nil, nil // exit 1: no such setting
	}
	var odd []string // keys with an =
	for line := range strings.Lines(string(out)) {
		scope, key, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || (scope != "local" && scope != "worktree") {
			continue
		}
		value := ""
		if key == "core.fsmonitor" {
			value = "false"
		}
		if strings.Contains(key, "=") {
			odd = append(odd, key)
			continue
		}
		args = append(args, "-c", key+"="+value)
	}
	if len(odd) == 0 {
		return args, nil, nil
	}
	if !configEnvWorks() {
		return nil, nil, errors.New("this git is older than 2.31 and cannot switch off the repository's filters")
	}
	first, _ := strconv.Atoi(os.Getenv("GIT_CONFIG_COUNT"))
	first = max(first, 0)
	env = []string{"GIT_CONFIG_COUNT=" + strconv.Itoa(first+len(odd))}
	for i, key := range odd {
		n := strconv.Itoa(first + i)
		env = append(env, "GIT_CONFIG_KEY_"+n+"="+key, "GIT_CONFIG_VALUE_"+n+"=")
	}
	return args, env, nil
}

// configEnvWorks tells whether git reads GIT_CONFIG_COUNT, which it does since 2.31. A test swaps it.
var configEnvWorks = sync.OnceValue(func() bool {
	out, err := exec.Command("git", "version").Output()
	if err != nil {
		return false
	}
	var major, minor int
	if _, err := fmt.Sscanf(strings.TrimPrefix(strings.TrimSpace(string(out)), "git version "), "%d.%d", &major, &minor); err != nil {
		return false
	}
	return major > 2 || major == 2 && minor >= 31
})
