package ui

import (
	"os"
	"testing"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// The tests assert English text whatever the locale of the machine running them.
func TestMain(m *testing.M) {
	i18n.SetLanguage("en")
	os.Exit(m.Run())
}
