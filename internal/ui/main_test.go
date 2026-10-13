package ui

import (
	"os"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
	"github.com/MathiasDrizzy/lazyfinder/internal/opener"
)

// The tests assert English text whatever the locale of the machine running them.
func TestMain(m *testing.M) {
	i18n.SetLanguage("en")
	opener.AppExists = func(string) bool { return false } // the menus must not depend on the apps installed here
	os.Exit(m.Run())
}
