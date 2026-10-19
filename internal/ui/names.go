package ui

import (
	"path/filepath"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// localName is the interface-language name of a standard place or folder, the way Finder shows it: Desktop reads
// Escritorio in Spanish although the folder on disk is still Desktop. Any other name is returned as it is.
func localName(name string) string {
	switch name {
	case "Home":
		return i18n.T("Inicio", "Home")
	case "Desktop":
		return i18n.T("Escritorio", "Desktop")
	case "Documents":
		return i18n.T("Documentos", "Documents")
	case "Downloads":
		return i18n.T("Descargas", "Downloads")
	case "Applications":
		return i18n.T("Aplicaciones", "Applications")
	case "Movies":
		return i18n.T("Películas", "Movies")
	case "Music":
		return i18n.T("Música", "Music")
	case "Pictures":
		return i18n.T("Imágenes", "Pictures")
	case "Public":
		return i18n.T("Público", "Public")
	case "Library":
		return i18n.T("Biblioteca", "Library")
	case "Trash":
		return i18n.T("Papelera", "Trash")
	case "Volumes":
		return i18n.T("Volúmenes", "Volumes")
	case "Computer":
		return i18n.T("Equipo", "Computer")
	}
	return name
}

// folderLabel is how a folder is named on screen: the standard folders of the home folder and /Applications are
// localized (as Finder does), everything else keeps the name it has on disk.
func (m *Model) folderLabel(path string) string {
	base := filepath.Base(path)
	if path == "/" {
		return "/"
	}
	if dir := filepath.Dir(path); dir == m.deps.Home || path == "/Applications" {
		return localName(base)
	}
	return shownName(base)
}
