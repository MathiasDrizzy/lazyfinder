package ui

import (
	"strings"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

func TestExplainGitTurnsWhatGitSaidIntoOneShortSentence(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	i18n.SetLanguage("en")
	cases := []struct {
		name, git string
		has       []string
		hasNot    []string
	}{
		{"one file blocks a checkout",
			"error: Your local changes to the following files would be overwritten by checkout: · \tmac/config/herdr/config.toml · Please commit your changes or stash them before you switch branches.",
			[]string{"config.toml", "commit or stash"}, []string{"error:", "Please", " · "}},
		{"two files", "error: Your local changes to the following files would be overwritten by checkout: · \ta.txt · \tb.txt",
			[]string{"a.txt", "1 more", "commit or stash"}, []string{"error:", "b.txt"}},
		{"a merge", "error: Your local changes to the following files would be overwritten by merge: · \tnotes.txt · Please commit your changes or stash them before you merge.",
			[]string{"notes.txt", "commit or stash"}, []string{"error:"}},
		{"fatal prefix goes", "fatal: not a git repository", []string{"not a git repository"}, []string{"fatal:"}},
		{"anything else is kept", "pathspec 'zz' did not match any file(s) known to git", []string{"pathspec 'zz' did not match"}, nil},
	}
	for _, c := range cases {
		got := explainGit(c.git)
		for _, w := range c.has {
			if !strings.Contains(got, w) {
				t.Errorf("%s: %q lacks %q", c.name, got, w)
			}
		}
		for _, w := range c.hasNot {
			if strings.Contains(got, w) {
				t.Errorf("%s: %q has %q", c.name, got, w)
			}
		}
	}
}

func TestExplainGitIsInSpanishToo(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	i18n.SetLanguage("es")
	got := explainGit("error: Your local changes to the following files would be overwritten by checkout: · \tnotes.txt · Please commit your changes or stash them before you switch branches.")
	if !strings.Contains(got, "notes.txt") || !strings.Contains(got, "stash") || strings.Contains(got, "overwritten") {
		t.Fatalf("got %q", got)
	}
}
