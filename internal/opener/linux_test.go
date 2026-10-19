package opener

import (
	"slices"
	"testing"
)

// On Linux there is no open(1) (on Debian it is openvt, which starts a virtual console) and no qlmanage: the default
// app and the Q fallback open the file with xdg-open, the desktop's way.
func TestOnLinuxTheDefaultAppIsXdgOpen(t *testing.T) {
	old := goos
	goos = "linux"
	t.Cleanup(func() { goos = old })
	for name, c := range map[string]Command{"default": systemDefault("/x/a.pdf"), "quicklook": QuickLook("/x/a.pdf")} {
		if !slices.Equal(c.Argv, []string{"xdg-open", "/x/a.pdf"}) {
			t.Errorf("%s argv = %q", name, c.Argv)
		}
	}
}

func TestOnMacOSTheDefaultAppIsOpen(t *testing.T) {
	old := goos
	goos = "darwin"
	t.Cleanup(func() { goos = old })
	if c := systemDefault("/x/a.pdf"); !slices.Equal(c.Argv, []string{"open", "--", "/x/a.pdf"}) {
		t.Errorf("argv = %q", c.Argv)
	}
	if c := QuickLook("/x/a.pdf"); !slices.Equal(c.Argv, []string{"qlmanage", "-p", "/x/a.pdf"}) {
		t.Errorf("quicklook argv = %q", c.Argv)
	}
}
