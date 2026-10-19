# Git operations you ask for run the repository's config

* Status: accepted
* Date: 2026-10-19

## Context and problem statement

lazyfinder reads the git status and diffs of any folder it shows, and a folder may come from anywhere (a zip, a
shared drive). Reading is hardened: the repository's own `core.fsmonitor` and filters are switched off, submodules are
not entered, and textconv and external diff drivers are off. The final security review of the audit asked whether the
operations the user asks for (stage, unstage, discard, stash, switch or create a branch, fetch, pull, push) should be
hardened the same way, since they run the same settings and `core.sshCommand`.

## Decision

They are not. An operation you ask for runs the repository's config exactly as `git` itself does when you type the
same command in that folder. lazyfinder adds nothing to what git runs: no shell, and paths go as literal pathspecs.

## Consequences

* A repository that needs its own filters to stage (`git lfs install --local`, git-crypt) keeps working from the Git
  tab, as it does in the terminal, lazygit or VS Code.
* Looking at a folder never runs its config; acting on it as a git repository trusts it, as git does. Discard asks
  first. `SECURITY.md` says both.
