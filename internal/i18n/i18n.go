// Package i18n translates the texts of the interface. The code writes each text as a Spanish/English pair,
// i18n.T("Notas", "Notes"); Spanish and English come from that pair and the other languages (pt, fr, de, it, ja, zh)
// from the catalogs in catalog/<language>.json, keyed by the English text. A text with no translation falls back to
// English; the coverage test (keys_test.go) keeps any from being left untranslated.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Language is a language of the interface, by its two-letter code.
type Language string

const (
	LangEN Language = "en"
	LangES Language = "es"
	LangPT Language = "pt" // Brazilian Portuguese (pt-BR)
	LangFR Language = "fr"
	LangDE Language = "de"
	LangIT Language = "it"
	LangJA Language = "ja"
	LangZH Language = "zh" // Simplified Chinese (zh-CN)
)

// Languages are the supported languages, in the order Settings cycles through them.
var Languages = []Language{LangEN, LangES, LangPT, LangFR, LangDE, LangIT, LangJA, LangZH}

// Name is the name of the language in that language (the one Settings shows).
func (l Language) Name() string {
	switch l {
	case LangES:
		return "Español"
	case LangPT:
		return "Português (BR)"
	case LangFR:
		return "Français"
	case LangDE:
		return "Deutsch"
	case LangIT:
		return "Italiano"
	case LangJA:
		return "日本語"
	case LangZH:
		return "简体中文"
	}
	return "English"
}

//go:embed catalog/*.json
var catalogFS embed.FS

var (
	catalogOnce sync.Once
	catalogs    map[Language]map[string]string
)

// catalog returns the translations of l (key: the English text), read once.
func catalog(l Language) map[string]string {
	catalogOnce.Do(func() {
		catalogs = map[Language]map[string]string{}
		for _, lang := range Languages {
			data, err := catalogFS.ReadFile("catalog/" + string(lang) + ".json")
			if err != nil {
				continue // en and es have no catalog
			}
			m := map[string]string{}
			if err := json.Unmarshal(data, &m); err != nil {
				panic("lazyfinder: the catalog of " + string(lang) + " is malformed: " + err.Error()) // a bug in the repository
			}
			catalogs[lang] = m
		}
	})
	return catalogs[l]
}

var (
	mu          sync.RWMutex
	currentLang = Detect(os.Getenv)
)

// Parse recognizes a language by its code (en, es, pt, pt-BR, pt_BR, zh-CN…) or by its name.
func Parse(s string) (Language, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if i := strings.IndexAny(s, ".@"); i >= 0 { // "pt_BR.UTF-8"
		s = s[:i]
	}
	if i := strings.IndexAny(s, "-_"); i >= 0 {
		s = s[:i]
	}
	switch s {
	case "en", "english", "ingles", "inglés":
		return LangEN, true
	case "es", "spanish", "español", "espanol":
		return LangES, true
	case "pt", "portuguese", "português", "portugues", "português (br)":
		return LangPT, true
	case "fr", "french", "français", "francais":
		return LangFR, true
	case "de", "german", "deutsch":
		return LangDE, true
	case "it", "italian", "italiano":
		return LangIT, true
	case "ja", "jp", "japanese", "日本語":
		return LangJA, true
	case "zh", "chinese", "中文", "简体中文":
		return LangZH, true
	}
	return LangEN, false
}

// Detect picks the system language from the locale variables, in POSIX order (LC_ALL, LC_MESSAGES, LANG): the first
// one set decides ("pt_BR.UTF-8" → pt). An unsupported language, or no variable at all: English.
func Detect(getenv func(string) string) Language {
	for _, name := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := strings.TrimSpace(getenv(name))
		if v == "" || v == "POSIX" || v == "C" || strings.HasPrefix(v, "C.") { // the "C" locale (C.UTF-8 in containers) picks no language
			continue
		}
		if l, ok := Parse(v); ok {
			return l
		}
		return LangEN
	}
	return LangEN
}

// SetLanguage sets the current language: a supported code or name, or "auto" (the system one). Anything else: English.
func SetLanguage(lang string) {
	l := LangEN
	if s := strings.ToLower(strings.TrimSpace(lang)); s == "auto" || s == "" {
		l = Detect(os.Getenv)
	} else if p, ok := Parse(s); ok {
		l = p
	}
	mu.Lock()
	currentLang = l
	mu.Unlock()
}

// CurrentLanguage returns the active language.
func CurrentLanguage() Language {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

// T returns the text in the active language: es and en come from the code, the other languages from the catalog
// (key: en), and a text with no translation falls back to English.
func T(es, en string) string {
	switch l := CurrentLanguage(); l {
	case LangES:
		return es
	case LangEN:
		return en
	default:
		if s := catalog(l)[en]; s != "" {
			return s
		}
		return en
	}
}

// E is T for the texts of errors: the same behavior, but the coverage test of the catalogs does not require them,
// because there are many and an untranslated language falls back to English, never to Spanish.
func E(spanish, english string) string {
	switch l := CurrentLanguage(); l {
	case LangES:
		return spanish
	case LangEN:
		return english
	default:
		if t := catalog(l)[english]; t != "" {
			return t
		}
		return english
	}
}

// textErr is an error whose text is chosen when it is shown, not when it is made: it serves the sentinel errors of the
// packages (`var ErrX = i18n.NewError(…)`), made before the language is known. Two with the same text are different
// (errors.Is compares the pointer).
type textErr struct{ es, en string }

func (e *textErr) Error() string { return E(e.es, e.en) }

// NewError makes an error written in the language of the interface when it is shown.
func NewError(spanish, english string) error { return &textErr{spanish, english} }

// Errorf is fmt.Errorf with the format in the language of the interface (%w works). The arguments go in the same order
// in both texts.
func Errorf(spanish, english string, args ...any) error {
	return fmt.Errorf(E(spanish, english), args...)
}
