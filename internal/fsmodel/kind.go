package fsmodel

import (
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// kinds maps lowercase extensions to the Finder-style kind name. A table keeps Kind deterministic and free of
// subprocesses (docs/adr/0002-architecture.md).
var kinds = map[string][2]string{ // extension -> {Spanish, English}
	"txt": {"Texto sin formato", "Plain Text"}, "text": {"Texto sin formato", "Plain Text"}, "md": {"Documento de Markdown", "Markdown Document"}, "markdown": {"Documento de Markdown", "Markdown Document"},
	"rtf": {"Documento de texto enriquecido", "Rich Text Document"}, "pdf": {"Documento de PDF", "PDF Document"}, "doc": {"Documento de Word", "Word Document"}, "docx": {"Documento de Word", "Word Document"},
	"xls": {"Hoja de cálculo de Excel", "Excel Spreadsheet"}, "xlsx": {"Hoja de cálculo de Excel", "Excel Spreadsheet"}, "csv": {"Documento de CSV", "CSV Document"}, "ppt": {"Presentación de PowerPoint", "PowerPoint Presentation"},
	"pptx": {"Presentación de PowerPoint", "PowerPoint Presentation"}, "key": {"Presentación de Keynote", "Keynote Presentation"}, "pages": {"Documento de Pages", "Pages Document"},
	"numbers": {"Hoja de cálculo de Numbers", "Numbers Spreadsheet"}, "odt": {"Texto de OpenDocument", "OpenDocument Text"}, "ods": {"Hoja de cálculo de OpenDocument", "OpenDocument Spreadsheet"},
	"jpg": {"Imagen JPEG", "JPEG image"}, "jpeg": {"Imagen JPEG", "JPEG image"}, "png": {"Imagen PNG", "PNG image"}, "gif": {"Imagen GIF", "GIF image"}, "webp": {"Imagen WebP", "WebP image"},
	"heic": {"Imagen HEIC", "HEIC image"}, "tif": {"Imagen TIFF", "TIFF image"}, "tiff": {"Imagen TIFF", "TIFF image"}, "bmp": {"Imagen BMP", "BMP image"}, "svg": {"Imagen SVG", "SVG image"},
	"ico": {"Icono", "Icon image"}, "mp3": {"Audio MP3", "MP3 audio"}, "wav": {"Audio WAV", "WAV audio"}, "flac": {"Audio FLAC", "FLAC audio"}, "m4a": {"Audio MPEG-4", "MPEG-4 audio"},
	"mp4": {"Película MPEG-4", "MPEG-4 movie"}, "mov": {"Película QuickTime", "QuickTime movie"}, "mkv": {"Vídeo Matroska", "Matroska video"}, "avi": {"Película AVI", "AVI movie"}, "webm": {"Vídeo WebM", "WebM video"},
	"zip": {"Archivo ZIP", "ZIP archive"}, "tar": {"Archivo TAR", "TAR archive"}, "gz": {"Archivo GZip", "GZip archive"}, "tgz": {"Archivo GZip", "GZip archive"}, "bz2": {"Archivo BZip2", "BZip2 archive"},
	"xz": {"Archivo XZ", "XZ archive"}, "7z": {"Archivo 7-Zip", "7-Zip archive"}, "rar": {"Archivo RAR", "RAR archive"}, "dmg": {"Imagen de disco", "Disk Image"}, "iso": {"Imagen de disco", "Disk Image"},
	"app": {"Aplicación", "Application"}, "pkg": {"Paquete de instalación", "Installer Package"}, "go": {"Código fuente de Go", "Go Source"}, "py": {"Script de Python", "Python Script"}, "js": {"JavaScript", "JavaScript"},
	"mjs": {"JavaScript", "JavaScript"}, "ts": {"Código fuente de TypeScript", "TypeScript Source"}, "tsx": {"Código fuente de TypeScript", "TypeScript Source"}, "jsx": {"JavaScript", "JavaScript"},
	"rs": {"Código fuente de Rust", "Rust Source"}, "c": {"Código fuente de C", "C Source"}, "h": {"Cabecera de C", "C Header"}, "cpp": {"Código fuente de C++", "C++ Source"}, "hpp": {"Cabecera de C++", "C++ Header"},
	"java": {"Código fuente de Java", "Java Source"}, "kt": {"Código fuente de Kotlin", "Kotlin Source"}, "swift": {"Código fuente de Swift", "Swift Source"}, "rb": {"Script de Ruby", "Ruby Script"}, "php": {"Script de PHP", "PHP Script"},
	"sh": {"Script de Shell", "Shell Script"}, "zsh": {"Script de Shell", "Shell Script"}, "bash": {"Script de Shell", "Shell Script"}, "lua": {"Script de Lua", "Lua Script"}, "sql": {"Código fuente de SQL", "SQL Source"},
	"html": {"Documento de HTML", "HTML document"}, "htm": {"Documento de HTML", "HTML document"}, "css": {"Hoja de estilos CSS", "CSS Stylesheet"}, "json": {"Documento de JSON", "JSON Document"},
	"yaml": {"Documento de YAML", "YAML Document"}, "yml": {"Documento de YAML", "YAML Document"}, "toml": {"Documento de TOML", "TOML Document"}, "xml": {"Documento de XML", "XML Document"},
	"ini": {"Archivo de configuración", "Configuration File"}, "conf": {"Archivo de configuración", "Configuration File"}, "log": {"Archivo de registro", "Log File"}, "tf": {"Código fuente de Terraform", "Terraform Source"},
	"lock": {"Archivo de bloqueo", "Lock File"}, "sum": {"Archivo de sumas de verificación", "Checksum File"}, "mod": {"Módulo de Go", "Go Module"}, "ttf": {"Fuente TrueType", "TrueType font"}, "otf": {"Fuente OpenType", "OpenType font"},
	"woff": {"Fuente web", "Web font"}, "woff2": {"Fuente web", "Web font"},
}

// KindOf returns the Finder-style kind of an entry. Symlinks are "Alias" whatever they point to.
func KindOf(name string, isDir, isLink bool) string {
	switch {
	case isLink:
		return i18n.T("Alias", "Alias")
	case isDir:
		return i18n.T("Carpeta", "Folder")
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(name), "."))
	if ext == "" {
		return i18n.T("Documento", "Document")
	}
	if k, ok := kinds[ext]; ok {
		return i18n.E(k[0], k[1])
	}
	return strings.ToUpper(ext) + i18n.T(" (archivo)", " File")
}
