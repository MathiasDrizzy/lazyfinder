package theme

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const userThemeJSON = `{
  "source": "https://example.com/night",
  "base": "#0a1014", "mantle": "#070b0d", "surface0": "#162128", "surface1": "#222e36",
  "overlay0": "#758a96", "text": "#cccfd1", "subtext0": "#a9b0b5", "peach": "#99c1dc",
  "mauve": "#b389a7", "teal": "#5ba1a3", "green": "#7b9f7e", "red": "#bf878c",
  "blue": "#7f97be", "yellow": "#a1966d"
}`

func writeTheme(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func problemText(p UserThemeProblem) string { return fmt.Sprintf(p.EN, p.Args...) }

func cleanUserThemes(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { LoadUserThemes(t.TempDir()) })
}

func TestBuiltInThemesAreComplete(t *testing.T) {
	names := ThemeNames()
	if len(names) != 14 {
		t.Fatalf("want 14 built-in themes, got %d: %v", len(names), names)
	}
	for _, n := range names {
		p, ok := Lookup(n)
		if !ok || p.Name != n {
			t.Fatalf("theme %q: Lookup = %v, %q", n, ok, p.Name)
		}
		for _, c := range paletteSlots(&p) {
			if *c.dst == nil {
				t.Errorf("theme %q has no %s color", n, c.key)
			}
		}
	}
}

func TestDefaultThemeExistsAndTerminalIsValid(t *testing.T) {
	if _, ok := Lookup(Default); !ok {
		t.Fatalf("default theme %q is not built in", Default)
	}
	if !Valid(Terminal) || Valid("no-such-theme") || !Valid(Default) {
		t.Fatal("Valid must accept terminal and built-in names only")
	}
	if got := Names()[0]; got != Terminal {
		t.Fatalf("Names()[0] = %q, want %q", got, Terminal)
	}
}

func TestUserThemeLoadsAfterBuiltInOnes(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	writeTheme(t, dir, "night.json", userThemeJSON)
	writeTheme(t, dir, "notes.txt", "not a theme")
	if problems := LoadUserThemes(dir); len(problems) != 0 {
		t.Fatalf("unexpected problems: %v", problems)
	}
	names := ThemeNames()
	if len(names) != 15 || names[14] != "night" || !Valid("night") {
		t.Fatalf("night should follow the 14 built-in themes: %v", names)
	}
}

func TestUserThemeProblemsAreReportedAndOthersStillLoad(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	writeTheme(t, dir, "good.json", userThemeJSON)
	writeTheme(t, dir, "badhex.json", strings.Replace(userThemeJSON, "#0a1014", "red", 1))
	writeTheme(t, dir, "missing.json", strings.Replace(userThemeJSON, `"base": "#0a1014", `, "", 1))
	writeTheme(t, dir, "typo.json", strings.Replace(userThemeJSON, `"base"`, `"bse"`, 1))
	writeTheme(t, dir, "nord.json", userThemeJSON)
	writeTheme(t, dir, "Bad Name.json", userThemeJSON)
	writeTheme(t, dir, "notjson.json", "{")
	problems := LoadUserThemes(dir)
	if !Valid("good") {
		t.Fatal("the good theme must load despite the others")
	}
	want := []string{"badhex", "missing", "typo", "nord", "Bad Name", "notjson"}
	if len(problems) != len(want) {
		t.Fatalf("want %d problems, got %d: %v", len(want), len(problems), problems)
	}
	var all string
	for _, p := range problems {
		all += problemText(p) + "\n"
	}
	for _, frag := range []string{`"bse"`, `"base" is missing`, "not #rrggbb", "built-in theme has that name", "only allows a-z"} {
		if !strings.Contains(all, frag) {
			t.Errorf("problems should mention %q:\n%s", frag, all)
		}
	}
}

func TestUserThemeNeverReadsAFIFOOrAHugeFile(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "pipe.json"), 0o600); err != nil {
		t.Skip("no FIFOs here:", err)
	}
	writeTheme(t, dir, "huge.json", strings.Repeat(" ", maxThemeBytes+1))
	problems := LoadUserThemes(dir)
	if len(problems) != 2 {
		t.Fatalf("want 2 problems, got %v", problems)
	}
}

func TestMissingThemesFolderIsNotAProblem(t *testing.T) {
	cleanUserThemes(t)
	if problems := LoadUserThemes(filepath.Join(t.TempDir(), "nope")); len(problems) != 0 {
		t.Fatalf("a missing folder is fine: %v", problems)
	}
}

func TestReloadingDropsThemesThatDisappeared(t *testing.T) {
	cleanUserThemes(t)
	dir := t.TempDir()
	writeTheme(t, dir, "night.json", userThemeJSON)
	LoadUserThemes(dir)
	LoadUserThemes(t.TempDir())
	if Valid("night") {
		t.Fatal("a theme from a previous load must not survive")
	}
}

func TestSafeTextEscapesControlCharacters(t *testing.T) {
	if got := safeText("plain"); got != "plain" {
		t.Fatalf("plain text unchanged, got %q", got)
	}
	if got := safeText("a\x1b[31mb"); strings.ContainsRune(got, 0x1b) {
		t.Fatalf("escape must be quoted, got %q", got)
	}
}

func TestReadableKeepsGoodColorsAndFixesWeakOnes(t *testing.T) {
	bg := color.RGBA{R: 0xef, G: 0xf1, B: 0xf5, A: 255}
	good := color.RGBA{A: 255}
	if got := Readable(good, bg, 3.1); got != color.Color(good) {
		t.Fatalf("a readable color changed: %v", got)
	}
	weak := color.RGBA{R: 0xac, G: 0xb0, B: 0xbe, A: 255}
	fixed := Readable(weak, bg, 3.1)
	if Contrast(fixed, bg) < 3.1 {
		t.Fatalf("still weak: %.2f", Contrast(fixed, bg))
	}
	dark := color.RGBA{R: 0x1e, G: 0x1e, B: 0x2e, A: 255}
	if Contrast(Readable(color.RGBA{R: 0x45, G: 0x47, B: 0x5a, A: 255}, dark, 3.1), dark) < 3.1 {
		t.Fatal("a dark background was not lifted toward white")
	}
}
