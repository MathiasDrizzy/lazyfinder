package ui

import "charm.land/lipgloss/v2"

// Styles never rely on colour alone: selection is reverse video plus a pointer, folders carry a ">" marker, and
// errors start with "Error:" (NO_COLOR and monochrome terminals keep every distinction).
type styles struct {
	selected func(string) string
	dim      func(string) string
	bold     func(string) string
	folder   func(string) string
	accent   func(string) string
	errText  func(string) string
	border   func(string) string
}

func newStyles(theme string) styles {
	blue := lipgloss.Color("12")
	if theme == "light" {
		blue = lipgloss.Color("4")
	}
	r := func(s lipgloss.Style) func(string) string { return func(t string) string { return s.Render(t) } }
	return styles{
		selected: r(lipgloss.NewStyle().Reverse(true)),
		dim:      r(lipgloss.NewStyle().Faint(true)),
		bold:     r(lipgloss.NewStyle().Bold(true)),
		folder:   r(lipgloss.NewStyle().Foreground(blue).Bold(true)),
		accent:   r(lipgloss.NewStyle().Foreground(blue)),
		errText:  r(lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)),
		border:   r(lipgloss.NewStyle().Faint(true)),
	}
}
