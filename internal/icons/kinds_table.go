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

// kindGlyphs maps a kind to its Nerd Font glyph and the palette role closest to the color the herdr sidebar gives it,
// so every theme recolors it (order 07, revision 2, S3: "debe todo cambiar con respecto al color del tema").
var kindGlyphs = map[fileKind]struct {
	glyph string
	role  Role
}{
	kDir:        {"\uf07b", Folder},
	kDirOpen:    {"\uf07c", Folder},
	kRust:       {"\ue7a8", Peach},
	kPython:     {"\ue73c", Blue},
	kJs:         {"\ue74e", Yellow},
	kTs:         {"\ue628", Blue},
	kReact:      {"\ue7ba", Teal},
	kJson:       {"\ue60b", Yellow},
	kMarkdown:   {"\uf48a", Teal},
	kHtml:       {"\ue736", Red},
	kCss:        {"\ue749", Blue},
	kConfig:     {"\ue615", Plain},
	kXml:        {"\uf121", Peach},
	kShell:      {"\uf489", Green},
	kPowerShell: {"\U000f0a0a", Blue},
	kCFamily:    {"\ue61d", Mauve},
	kCSharp:     {"\U000f031b", Green},
	kGo:         {"\ue627", Teal},
	kRuby:       {"\ue791", Red},
	kPhp:        {"\ue73d", Mauve},
	kJava:       {"\ue738", Peach},
	kKotlin:     {"\ue634", Mauve},
	kSwift:      {"\ue755", Peach},
	kLua:        {"\ue620", Blue},
	kSql:        {"\ue706", Peach},
	kData:       {"\uf1c3", Green},
	kText:       {"\uf15c", Plain},
	kLog:        {"\uf15c", Plain},
	kPdf:        {"\uf1c1", Red},
	kImage:      {"\uf1c5", Teal},
	kAudio:      {"\uf1c7", Mauve},
	kVideo:      {"\uf1c8", Peach},
	kArchive:    {"\uf1c6", Yellow},
	kLock:       {"\uf023", Yellow},
	kBinary:     {"\uf471", Red},
	kFont:       {"\uf031", Plain},
	kNotebook:   {"\uf02d", Peach},
	kGit:        {"\ue702", Red},
	kDocker:     {"\uf308", Blue},
	kPackage:    {"\uf487", Peach},
	kBuild:      {"\uf0ad", Plain},
	kReadme:     {"\uf02d", Blue},
	kLicense:    {"\uf24e", Yellow},
	kEnvKey:     {"\uf084", Yellow},
	kFile:       {"\uf15b", Plain},
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
