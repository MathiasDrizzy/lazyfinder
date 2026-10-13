package icons

import "strings"

// kindOf classifies a file the way the herdr-sidebar plugin does (MIT, see NOTICE): whole names first, then the
// extension. ok is false for a file neither knows, so the caller can fall back to the wider extension table.
func kindOf(name string) (fileKind, bool) {
	lower := strings.ToLower(name)
	switch {
	case lower == "cargo.lock" || lower == "package-lock.json" || lower == "yarn.lock" || lower == "pnpm-lock.yaml" || lower == "go.sum":
		return kLock, true
	case lower == "cargo.toml" || lower == "package.json" || lower == "pyproject.toml" || lower == "go.mod" || lower == "gemfile":
		return kPackage, true
	case lower == "makefile" || lower == "justfile" || lower == "cmakelists.txt":
		return kBuild, true
	case lower == ".gitignore" || lower == ".gitattributes" || lower == ".gitmodules":
		return kGit, true
	case strings.HasPrefix(lower, "dockerfile") || strings.HasPrefix(lower, "docker-compose"):
		return kDocker, true
	case strings.HasPrefix(lower, "readme"):
		return kReadme, true
	case strings.HasPrefix(lower, "license") || lower == "copying":
		return kLicense, true
	case lower == ".env" || strings.HasPrefix(lower, ".env."):
		return kEnvKey, true
	}
	i := strings.LastIndexByte(lower, '.')
	if i < 0 {
		return kFile, false
	}
	k, ok := extKinds[lower[i+1:]]
	return k, ok
}

var extKinds = map[string]fileKind{}

func init() {
	for k, exts := range map[fileKind]string{
		kRust: "rs", kPython: "py pyi", kJs: "js mjs cjs", kTs: "ts", kReact: "jsx tsx", kJson: "json jsonc",
		kMarkdown: "md markdown", kHtml: "html htm", kCss: "css scss sass less",
		kConfig: "toml yaml yml ini cfg conf", kXml: "xml", kShell: "sh bash zsh fish",
		kPowerShell: "ps1 psm1 psd1 bat cmd", kCFamily: "c h cpp cc cxx hpp hh", kCSharp: "cs", kGo: "go",
		kRuby: "rb", kPhp: "php", kJava: "java jar", kKotlin: "kt kts", kSwift: "swift", kLua: "lua",
		kSql: "sql db sqlite sqlite3", kData: "csv tsv", kText: "txt", kLog: "log", kPdf: "pdf",
		kImage: "png jpg jpeg gif webp bmp ico svg tiff", kAudio: "mp3 wav flac ogg", kVideo: "mp4 mkv avi mov webm",
		kArchive: "zip tar gz tgz bz2 xz 7z rar", kLock: "lock", kBinary: "exe dll so dylib a o bin wasm",
		kFont: "ttf otf woff woff2", kNotebook: "ipynb",
	} {
		for _, e := range strings.Fields(exts) {
			extKinds[e] = k
		}
	}
}
