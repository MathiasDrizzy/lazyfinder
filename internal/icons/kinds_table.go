// The kinds, glyphs, colors and emoji below are those of the icon table of the herdr-sidebar plugin
// (https://github.com/alexarthurs/herdr-sidebar, MIT licence, see NOTICE).

package icons

type fileKind int

const (
	kDir fileKind = iota
	kDirOpen
	kRust
	kPython
	kJs
	kTs
	kReact
	kJson
	kMarkdown
	kHtml
	kCss
	kConfig
	kXml
	kShell
	kPowerShell
	kCFamily
	kCSharp
	kGo
	kRuby
	kPhp
	kJava
	kKotlin
	kSwift
	kLua
	kSql
	kData
	kText
	kLog
	kPdf
	kImage
	kAudio
	kVideo
	kArchive
	kLock
	kBinary
	kFont
	kNotebook
	kGit
	kDocker
	kPackage
	kBuild
	kReadme
	kLicense
	kEnvKey
	kFile
)

// kindGlyphs maps a kind to its Nerd Font glyph and its color (0xRRGGBB).
var kindGlyphs = map[fileKind]struct {
	glyph string
	rgb   uint32
}{
	kDir:        {"\uf07b", 0x90a4ae},
	kDirOpen:    {"\uf07c", 0x90a4ae},
	kRust:       {"\ue7a8", 0xdea584},
	kPython:     {"\ue73c", 0x3572a5},
	kJs:         {"\ue74e", 0xf1e05a},
	kTs:         {"\ue628", 0x3178c6},
	kReact:      {"\ue7ba", 0x61dafb},
	kJson:       {"\ue60b", 0xcbcb41},
	kMarkdown:   {"\uf48a", 0x519aba},
	kHtml:       {"\ue736", 0xe34c26},
	kCss:        {"\ue749", 0x42a5f5},
	kConfig:     {"\ue615", 0x6d8086},
	kXml:        {"\uf121", 0xe37933},
	kShell:      {"\uf489", 0x4eaa25},
	kPowerShell: {"\U000f0a0a", 0x5391fe},
	kCFamily:    {"\ue61d", 0xf34b7d},
	kCSharp:     {"\U000f031b", 0x178600},
	kGo:         {"\ue627", 0x00add8},
	kRuby:       {"\ue791", 0x701516},
	kPhp:        {"\ue73d", 0x4f5d95},
	kJava:       {"\ue738", 0xb07219},
	kKotlin:     {"\ue634", 0xa97bff},
	kSwift:      {"\ue755", 0xf05138},
	kLua:        {"\ue620", 0x51a0cf},
	kSql:        {"\ue706", 0xf29111},
	kData:       {"\uf1c3", 0x33a852},
	kText:       {"\uf15c", 0x9e9e9e},
	kLog:        {"\uf15c", 0x757575},
	kPdf:        {"\uf1c1", 0xe53935},
	kImage:      {"\uf1c5", 0x26a69a},
	kAudio:      {"\uf1c7", 0xec407a},
	kVideo:      {"\uf1c8", 0xff7043},
	kArchive:    {"\uf1c6", 0xafb42b},
	kLock:       {"\uf023", 0xffd54f},
	kBinary:     {"\uf471", 0xef5350},
	kFont:       {"\uf031", 0xb0bec5},
	kNotebook:   {"\uf02d", 0xf57c00},
	kGit:        {"\ue702", 0xf14e32},
	kDocker:     {"\uf308", 0x0db7ed},
	kPackage:    {"\uf487", 0x8d6e63},
	kBuild:      {"\uf0ad", 0x6d8086},
	kReadme:     {"\uf02d", 0x42a5f5},
	kLicense:    {"\uf24e", 0xffd54f},
	kEnvKey:     {"\uf084", 0xffd54f},
	kFile:       {"\uf15b", 0x90a4ae},
}

// kindEmoji maps a kind to its emoji.
var kindEmoji = map[fileKind]string{
	kDir:        "📁",
	kDirOpen:    "📂",
	kRust:       "🦀",
	kPython:     "🐍",
	kJs:         "🟨",
	kTs:         "🔷",
	kReact:      "🟦",
	kJson:       "🧾",
	kMarkdown:   "📝",
	kHtml:       "🌐",
	kCss:        "🎨",
	kConfig:     "🔧",
	kXml:        "📰",
	kShell:      "🐚",
	kPowerShell: "💻",
	kCFamily:    "🔩",
	kCSharp:     "🟣",
	kGo:         "🐹",
	kRuby:       "💎",
	kPhp:        "🐘",
	kJava:       "☕",
	kKotlin:     "🟪",
	kSwift:      "🐦",
	kLua:        "🌙",
	kSql:        "💾",
	kData:       "📊",
	kText:       "📄",
	kLog:        "📋",
	kPdf:        "📕",
	kImage:      "📷",
	kAudio:      "🎵",
	kVideo:      "🎬",
	kArchive:    "🧳",
	kLock:       "🔒",
	kBinary:     "⚡",
	kFont:       "🔤",
	kNotebook:   "📓",
	kGit:        "🙈",
	kDocker:     "🐳",
	kPackage:    "📦",
	kBuild:      "🔨",
	kReadme:     "📖",
	kLicense:    "📜",
	kEnvKey:     "🔑",
	kFile:       "📄",
}
