package fsmodel

import (
	"path/filepath"
	"strings"

	"github.com/MathiasDrizzy/lazyfinder/internal/i18n"
)

// kinds maps lowercase extensions to the Finder-style kind name. A table keeps Kind deterministic and free of
// subprocesses (docs/adr/0002-architecture.md).
var kinds = map[string]string{
	"txt": "Plain Text", "text": "Plain Text", "md": "Markdown Document", "markdown": "Markdown Document",
	"rtf": "Rich Text Document", "pdf": "PDF Document", "doc": "Word Document", "docx": "Word Document",
	"xls": "Excel Spreadsheet", "xlsx": "Excel Spreadsheet", "csv": "CSV Document", "ppt": "PowerPoint Presentation",
	"pptx": "PowerPoint Presentation", "key": "Keynote Presentation", "pages": "Pages Document",
	"numbers": "Numbers Spreadsheet", "odt": "OpenDocument Text", "ods": "OpenDocument Spreadsheet",
	"jpg": "JPEG image", "jpeg": "JPEG image", "png": "PNG image", "gif": "GIF image", "webp": "WebP image",
	"heic": "HEIC image", "tif": "TIFF image", "tiff": "TIFF image", "bmp": "BMP image", "svg": "SVG image",
	"ico": "Icon image", "mp3": "MP3 audio", "wav": "WAV audio", "flac": "FLAC audio", "m4a": "MPEG-4 audio",
	"mp4": "MPEG-4 movie", "mov": "QuickTime movie", "mkv": "Matroska video", "avi": "AVI movie", "webm": "WebM video",
	"zip": "ZIP archive", "tar": "TAR archive", "gz": "GZip archive", "tgz": "GZip archive", "bz2": "BZip2 archive",
	"xz": "XZ archive", "7z": "7-Zip archive", "rar": "RAR archive", "dmg": "Disk Image", "iso": "Disk Image",
	"app": "Application", "pkg": "Installer Package", "go": "Go Source", "py": "Python Script", "js": "JavaScript",
	"mjs": "JavaScript", "ts": "TypeScript Source", "tsx": "TypeScript Source", "jsx": "JavaScript",
	"rs": "Rust Source", "c": "C Source", "h": "C Header", "cpp": "C++ Source", "hpp": "C++ Header",
	"java": "Java Source", "kt": "Kotlin Source", "swift": "Swift Source", "rb": "Ruby Script", "php": "PHP Script",
	"sh": "Shell Script", "zsh": "Shell Script", "bash": "Shell Script", "lua": "Lua Script", "sql": "SQL Source",
	"html": "HTML document", "htm": "HTML document", "css": "CSS Stylesheet", "json": "JSON Document",
	"yaml": "YAML Document", "yml": "YAML Document", "toml": "TOML Document", "xml": "XML Document",
	"ini": "Configuration File", "conf": "Configuration File", "log": "Log File", "tf": "Terraform Source",
	"lock": "Lock File", "sum": "Checksum File", "mod": "Go Module", "ttf": "TrueType font", "otf": "OpenType font",
	"woff": "Web font", "woff2": "Web font",
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
		return k
	}
	return strings.ToUpper(ext) + i18n.T(" (archivo)", " File")
}
