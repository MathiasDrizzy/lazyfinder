package icons

import "testing"

func TestAHiddenGitFileKeepsItsRedColor(t *testing.T) {
	s := Set{Style: Nerd}
	for _, name := range []string{".gitignore", ".gitattributes"} {
		ic := s.Pick(Item{Name: name})
		if ic.Dim || ic.Role != Red {
			t.Errorf("%s: icon %+v, want the git color and not dimmed", name, ic)
		}
	}
	if ic := s.Pick(Item{Name: ".claude", IsDir: true}); !ic.Dim {
		t.Errorf("a hidden folder stays dimmed, got %+v", ic)
	}
}
