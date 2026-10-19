package ui

import (
	"regexp"
	"testing"
)

var sgrOnly = regexp.MustCompile(`^\x1b\[[0-9;:]*m`)

// Whatever a file name or a file's text holds, the frame keeps no escape sequence but colors and no control
// character but the line break, and sanitizing it again changes nothing.
func FuzzSafeFrame(f *testing.F) {
	for _, s := range []string{"plain", "\x1b[31mred\x1b[0m", "\x1b]52;c;aGk=\a", "\x1b[2J\x1b[?1049l", "\x1bc\x1b7\x1b(0", "\x1bP+q\x1b\\", "\x1b[", "\x1b", "a\tb\rc\x7f", "\xc2\x9b31m", "\x1b[38;2;1;2;3m", "日本\x1b]0;t"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := safeFrame(s)
		for i := 0; i < len(out); i++ {
			c := out[i]
			switch {
			case c == '\x1b':
				m := sgrOnly.FindString(out[i:])
				if m == "" {
					t.Fatalf("an escape that is not SGR survived at %d: %q from %q", i, out, s)
				}
				i += len(m) - 1
			case c < 0x20 && c != '\n', c == 0x7f:
				t.Fatalf("control 0x%02x survived: %q from %q", c, out, s)
			case c == 0xc2 && i+1 < len(out) && out[i+1] >= 0x80 && out[i+1] < 0xa0:
				t.Fatalf("a C1 control survived: %q from %q", out, s)
			}
		}
		if again := safeFrame(out); again != out {
			t.Fatalf("not idempotent: %q then %q", out, again)
		}
	})
}
