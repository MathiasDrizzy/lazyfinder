package icons

import "testing"

// The file kinds and glyphs are those of the herdr sidebar plugin (MIT, see NOTICE); each color is the palette role
// closest to herdr's, so the theme recolors it (order 07, revision 2, S3).
func TestFileKindsUseTheHerdrGlyphsAndColors(t *testing.T) {
	set := Set{Style: Nerd}
	for _, c := range []struct {
		name  string
		glyph string
		role  Role
	}{
		{"main.rs", "", Peach},
		{"MAIN.GO", "", Teal},
		{".gitignore", "", Red},
		{"README.md", "", Blue},
		{"go.mod", "", Peach},
		{"go.sum", "", Yellow},
		{"Dockerfile", "", Blue},
		{".env.local", "", Yellow},
		{"photo.JPG", "", Teal},
		{"Makefile", "", Plain},
		{"LICENSE", "", Yellow},
		{"data.csv", "", Green},
		{"script.sh", "", Green},
	} {
		ic := set.Pick(Item{Name: c.name})
		if ic.Glyph != c.glyph || ic.Role != c.role {
			t.Errorf("%s = %q role %d, want %q role %d", c.name, ic.Glyph, ic.Role, c.glyph, c.role)
		}
	}
}

func TestTheEmojiThemeDrawsOneColoredEmojiPerKindWithoutColors(t *testing.T) {
	set := Set{Style: Emoji}
	if set.Width() != 3 {
		t.Fatalf("an emoji is two cells wide plus the space: Width = %d", set.Width())
	}
	for name, want := range map[string]string{"main.rs": "🦀", "a.py": "🐍", "Dockerfile": "🐳", ".gitignore": "🙈", "a.unknownext": "📄"} {
		ic := set.Pick(Item{Name: name})
		if ic.Glyph != want || !ic.IsEmoji {
			t.Errorf("%s = %q (emoji %v), want the emoji %q with its own colors", name, ic.Glyph, ic.IsEmoji, want)
		}
	}
	if g := set.Pick(Item{Name: "src", IsDir: true, Open: true}).Glyph; g != "📂" {
		t.Errorf("open folder = %q", g)
	}
}

func TestTheUserMapCanNameAWholeFile(t *testing.T) {
	set := Set{Style: Nerd, Map: map[string]string{"Makefile": "M", ".gitignore": "G"}}
	if g := set.Pick(Item{Name: "makefile"}).Glyph; g != "M" {
		t.Errorf("makefile = %q", g)
	}
	if g := set.Pick(Item{Name: ".gitignore"}).Glyph; g != "G" {
		t.Errorf(".gitignore = %q", g)
	}
}
