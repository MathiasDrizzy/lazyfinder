package git

import (
	"context"
	"os"
	"strconv"
	"strings"
)

// commandKeys are the settings that make git run a program while it only reads the working tree: the fsmonitor hook
// on status and the clean, smudge and process filters on any file git has to compare. textconv is turned off with
// --no-textconv where a diff is shown, and external diff drivers with --no-ext-diff.
const commandKeys = `^(core\.fsmonitor|filter\..+\.(clean|smudge|process))$`

// readOnly returns the environment that switches off every commandKeys setting the repository itself defines, in
// .git/config, a file it includes or a worktree config. A folder you only look at may come from anywhere (a zip, a
// shared drive), and looking must not run what its config says. The same settings from your global or system config,
// or from the command line, are yours and stay on, so git-lfs and an fsmonitor daemon set up there keep working.
// Asking git config runs nothing. The settings go as GIT_CONFIG_KEY_n and GIT_CONFIG_VALUE_n, after any the
// environment already has: -c splits at the first =, so -c filter.x=y.clean= set filter.x and left a filter named
// "x=y" on. A submodule has a config of its own that this does not read, so the reads pass
// --ignore-submodules=dirty: git then compares the submodule's commit and never runs a status inside it.
func readOnly(ctx context.Context, dir string) []string {
	out, err := git(ctx, dir, "config", "--includes", "--show-scope", "--name-only", "--get-regexp", commandKeys)
	if err != nil {
		return nil // exit 1: no such setting
	}
	var keys []string
	for line := range strings.Lines(string(out)) {
		scope, key, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || (scope != "local" && scope != "worktree") {
			continue
		}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	first, _ := strconv.Atoi(os.Getenv("GIT_CONFIG_COUNT"))
	first = max(first, 0)
	env := []string{"GIT_CONFIG_COUNT=" + strconv.Itoa(first+len(keys))}
	for i, key := range keys {
		value := ""
		if key == "core.fsmonitor" {
			value = "false"
		}
		n := strconv.Itoa(first + i)
		env = append(env, "GIT_CONFIG_KEY_"+n+"="+key, "GIT_CONFIG_VALUE_"+n+"="+value)
	}
	return env
}
