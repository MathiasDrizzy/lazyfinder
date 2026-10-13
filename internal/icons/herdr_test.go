package icons

import "testing"

// The file kinds, glyphs and colors are those of the herdr sidebar plugin (MIT, see NOTICE).
func TestFileKindsUseTheHerdrGlyphsAndColors(t *testing.T) {
	set := Set{Style: Nerd}
	for _, c := range []struct {
		name  string
		glyph string
		rgb   uint32
	}{
		{"main.rs", "", 0xdea584},
		{"MAIN.GO", "", 0x00add8},
		{".gitignore", "", 0xf14e32},
		{"README.md", "", 0x42a5f5},
		{"go.mod", "", 0x8d6e63},
		{"go.sum", "", 0xffd54f},
		{"Dockerfile", "", 0x0db7ed},
		{".env.local", "", 0xffd54f},
		{"photo.JPG", "", 0x26a69a},
		{"Makefile", "", 0x6d8086},
		{"LICENSE", "", 0xffd54f},
		{"data.csv", "", 0x33a852},
		{"script.sh", "", 0x4eaa25},
	} {
		ic := set.Pick(Item{Name: c.name})
		if ic.Glyph != c.glyph || !ic.HasRGB || ic.RGB != c.rgb {
			t.Errorf("%s = %q %#06x (rgb %v), want %q %#06x", c.name, ic.Glyph, ic.RGB, ic.HasRGB, c.glyph, c.rgb)
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
		if ic.Glyph != want || ic.HasRGB {
			t.Errorf("%s = %q (rgb %v), want %q without a color", name, ic.Glyph, ic.HasRGB, want)
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
