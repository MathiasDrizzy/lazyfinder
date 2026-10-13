package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/config"
	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

func TestSidebarAndTitlesUseTheInterfaceLanguage(t *testing.T) {
	t.Cleanup(func() { i18n.SetLanguage("en") })
	e := newEnv(t)
	docs := filepath.Join(e.home, "Documents")
	if err := os.Mkdir(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	d := e.m.deps
	d.Home, d.Start = e.home, docs
	d.Cfg.Sidebar = config.Sidebar{
		Favorites: []config.Place{{Name: "Home", Path: e.home}, {Name: "Documents", Path: docs}, {Name: "My stuff", Path: e.root}},
		Locations: []config.Place{{Name: "Volumes", Path: "/Volumes"}},
	}
	for lang, want := range map[string][]string{"es": {"Inicio", "Documentos", "Volúmenes"}, "fr": {"Accueil", "Documents", "Volumes"}, "en": {"Home", "Documents", "Volumes"}} {
		i18n.SetLanguage(lang)
		m := New(d)
		got := map[string]bool{}
		for _, p := range m.places() {
			got[p.name] = true
		}
		for _, w := range want {
			if !got[w] {
				t.Errorf("%s: sidebar lacks %q: %v", lang, w, got)
			}
		}
		if !got["My stuff"] {
			t.Errorf("%s: a name the user chose must stay as it is", lang)
		}
		if title := m.folderTitle(); title != want[1] {
			t.Errorf("%s: title = %q, want %q", lang, title, want[1])
		}
	}
}
