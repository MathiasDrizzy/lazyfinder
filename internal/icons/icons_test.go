package icons

import "testing"

func TestEveryKindHasItsOwnGlyph(t *testing.T) {
	set := Set{Style: Nerd}
	closed := set.Pick(Item{Name: "src", IsDir: true})
	open := set.Pick(Item{Name: "src", IsDir: true, Open: true})
	if closed.Glyph == "" || open.Glyph == "" || closed.Glyph == open.Glyph {
		t.Fatalf("a folder needs a closed and an open glyph: %q %q", closed.Glyph, open.Glyph)
	}
	seen := map[string]string{}
	for _, name := range []string{"main.go", "a.py", "a.js", "a.ts", "a.rs", "a.md", "a.json", "a.png", "a.mp3", "a.mp4", "a.zip", "a.pdf", "a.docx", "a.xlsx", "a.pptx", "a.sh", "a.html", "a.css", "a.dmg", "Safari.app"} {
		g := set.Pick(Item{Name: name, IsDir: name == "Safari.app"}).Glyph
		if g == "" {
			t.Errorf("%s has no glyph", name)
		}
		seen[g] = name
	}
	if len(seen) < 15 {
		t.Errorf("only %d different glyphs for 20 kinds: %v", len(seen), seen)
	}
	if set.Pick(Item{Name: "x"}).Glyph == "" || set.Pick(Item{Name: "x.unknownext"}).Glyph == "" {
		t.Error("an unknown file still needs the generic glyph")
	}
}

func TestLinksAndHiddenFilesAreRecognized(t *testing.T) {
	set := Set{Style: Nerd}
	file := set.Pick(Item{Name: "a.txt"})
	if link := set.Pick(Item{Name: "a.txt", IsLink: true}); link.Glyph == file.Glyph {
		t.Error("a link must not look like the file")
	}
	if hid := set.Pick(Item{Name: ".env"}); !hid.Dim {
		t.Error("a hidden file is drawn dim")
	}
}

func TestUserMapOverridesAnExtension(t *testing.T) {
	set := Set{Style: Nerd, Map: map[string]string{"go": "G", ".MD": "M"}}
	if g := set.Pick(Item{Name: "main.go"}).Glyph; g != "G" {
		t.Errorf("go = %q", g)
	}
	if g := set.Pick(Item{Name: "README.md"}).Glyph; g != "M" {
		t.Errorf("md (dot and case ignored) = %q", g)
	}
}

func TestAsciiAndOffNeedNoNerdFont(t *testing.T) {
	for _, g := range []string{
		Set{Style: ASCII}.Pick(Item{Name: "d", IsDir: true}).Glyph,
		Set{Style: ASCII}.Pick(Item{Name: "f.go"}).Glyph,
	} {
		for _, r := range g {
			if r > 0x7e {
				t.Errorf("ascii glyph %q is not ASCII", g)
			}
		}
	}
	if set := (Set{Style: Off}); set.Width() != 0 || set.Pick(Item{Name: "a"}).Glyph != "" {
		t.Error("off draws nothing")
	}
	if (Set{Style: Nerd}).Width() != 2 || (Set{Style: ASCII}).Width() != 2 {
		t.Error("a glyph takes one cell and a space")
	}
}
