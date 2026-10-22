// Command caskfix rewrites the postflight block GoReleaser v2.18 writes into the Homebrew cask as postflight_steps,
// which Homebrew does not warn about. It reads the cask on stdin and writes it on stdout. Remove it once GoReleaser
// writes postflight_steps itself (hooks.post.install_steps, goreleaser/goreleaser#6873, planned for v2.19).
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// goreleaserBlock is the block .goreleaser.yaml (hooks.post.install) makes GoReleaser write.
const goreleaserBlock = `  postflight do
    if OS.mac?
      system_command "/usr/bin/xattr",
                     args: ["-dr", "com.apple.quarantine", "#{staged_path}/lazyfinder"]
    end
  end
`

// stepsBlock does the same: on a Mac it removes the quarantine of the binary, and a failure does not stop the
// install.
const stepsBlock = `  postflight_steps do
    run "/usr/bin/xattr", args: ["-dr", "com.apple.quarantine", "{{staged_path}}/lazyfinder"], must_succeed: false
  end
`

// fix returns cask with the GoReleaser block replaced, and whether it changed. Any other postflight block is an
// error: it is not guessed at.
func fix(cask string) (string, bool, error) {
	if strings.Contains(cask, goreleaserBlock) {
		return strings.Replace(cask, goreleaserBlock, stepsBlock, 1), true, nil
	}
	if strings.Contains(cask, "  postflight do\n") {
		return "", false, errors.New("the cask has a postflight block that is not the one GoReleaser writes")
	}
	return cask, false, nil
}

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "caskfix:", err)
		os.Exit(1)
	}
	out, changed, err := fix(string(in))
	if err != nil {
		fmt.Fprintln(os.Stderr, "caskfix:", err)
		os.Exit(1)
	}
	if !changed {
		fmt.Fprintln(os.Stderr, "caskfix: nothing to change")
	}
	fmt.Print(out)
}
