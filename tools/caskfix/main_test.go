package main

import (
	"strings"
	"testing"
)

// goreleaserCask is the cask GoReleaser v2.18 writes: its postflight block makes Homebrew warn "Calling `postflight`
// is deprecated! Use `postflight_steps` instead" on every brew command (order 2026-10-22-17).
const goreleaserCask = `cask "lazyfinder" do
  binary "lazyfinder"

  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr",
                     args: ["-dr", "com.apple.quarantine", "#{staged_path}/lazyfinder"]
    end
  end

  # No zap stanza required
end
`

const fixedCask = `cask "lazyfinder" do
  binary "lazyfinder"

  postflight_steps do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/lazyfinder"], must_succeed: false
  end

  # No zap stanza required
end
`

func TestThePostflightBlockBecomesPostflightSteps(t *testing.T) {
	got, changed, err := fix(goreleaserCask)
	if err != nil || !changed || got != fixedCask {
		t.Fatalf("changed %v, err %v:\n%s", changed, err, got)
	}
}

func TestAFixedCaskStaysAsItIs(t *testing.T) {
	if got, changed, err := fix(fixedCask); err != nil || changed || got != fixedCask {
		t.Fatalf("changed %v, err %v:\n%s", changed, err, got)
	}
}

// A postflight block that is not the one GoReleaser writes is not guessed at: the release step fails and says so.
func TestAnUnknownPostflightBlockIsAnError(t *testing.T) {
	other := strings.Replace(goreleaserCask, "com.apple.quarantine", "com.example.other", 1)
	if _, _, err := fix(other); err == nil {
		t.Fatal("an unknown postflight block must not be rewritten")
	}
}
