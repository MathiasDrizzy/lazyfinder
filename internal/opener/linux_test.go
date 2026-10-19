package opener

import (
	"slices"
	"testing"
)

// On Linux there is no open(1) (on Debian it is openvt, which starts a virtual console) and no qlmanage: the default
// app and the Q fallback open the file with xdg-open, the desktop's way.
// useGOOS makes the commands the ones of system g for one test.
func useGOOS(t *testing.T, g string) {
	t.Helper()
	old := goos
	goos = g
	t.Cleanup(func() { goos = old })
}

func TestOnLinuxTheDefaultAppIsXdgOpen(t *testing.T) {
	useGOOS(t, "linux")
	for name, c := range map[string]Command{"default": systemDefault("/x/a.pdf"), "quicklook": QuickLook("/x/a.pdf")} {
		if !slices.Equal(c.Argv, []string{"xdg-open", "/x/a.pdf"}) {
			t.Errorf("%s argv = %q", name, c.Argv)
		}
	}
}

func TestOnMacOSTheDefaultAppIsOpen(t *testing.T) {
	useGOOS(t, "darwin")
	if c := systemDefault("/x/a.pdf"); !slices.Equal(c.Argv, []string{"open", "--", "/x/a.pdf"}) {
		t.Errorf("argv = %q", c.Argv)
	}
	if c := QuickLook("/x/a.pdf"); !slices.Equal(c.Argv, []string{"qlmanage", "-p", "/x/a.pdf"}) {
		t.Errorf("quicklook argv = %q", c.Argv)
	}
}
