package git

import (
	"context"
	"strings"
)

// commandKeys are the settings that make git run a program while it only reads the working tree: the fsmonitor hook
// on status and the clean, smudge and process filters on any file git has to compare. textconv is turned off with
// --no-textconv where a diff is shown, and external diff drivers with --no-ext-diff.
const commandKeys = `^(core\.fsmonitor|filter\..+\.(clean|smudge|process))$`

// readOnly returns the -c options that switch off every commandKeys setting the repository itself defines, in
// .git/config, a file it includes or a worktree config. A folder you only look at may come from anywhere (a zip, a
// shared drive), and looking must not run what its config says. The same settings from your global or system config,
// or from the command line, are yours and stay on, so git-lfs and an fsmonitor daemon set up there keep working.
// Asking git config runs nothing. A submodule has a config of its own that this does not read, so the reads pass
// --ignore-submodules=dirty: git then compares the submodule's commit and never runs a status inside it.
func readOnly(ctx context.Context, dir string) []string {
	out, err := git(ctx, dir, "config", "--includes", "--show-scope", "--name-only", "--get-regexp", commandKeys)
	if err != nil {
		return nil // exit 1: no such setting
	}
	var opts []string
	for line := range strings.Lines(string(out)) {
		scope, key, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok || (scope != "local" && scope != "worktree") {
			continue
		}
		value := ""
		if key == "core.fsmonitor" {
			value = "false"
		}
		opts = append(opts, "-c", key+"="+value)
	}
	return opts
}
