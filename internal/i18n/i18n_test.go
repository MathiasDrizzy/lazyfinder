package i18n

import "testing"

func TestParseAcceptsCodesLocalesAndNames(t *testing.T) {
	cases := map[string]Language{
		"en": LangEN, "es": LangES, "pt_BR.UTF-8": LangPT, "pt-BR": LangPT, "fr": LangFR, "de_DE": LangDE,
		"it": LangIT, "ja_JP.UTF-8": LangJA, "zh-CN": LangZH, "Español": LangES, "日本語": LangJA,
	}
	for in, want := range cases {
		if got, ok := Parse(in); !ok || got != want {
			t.Errorf("Parse(%q) = %v %v, want %v", in, got, ok, want)
		}
	}
	if _, ok := Parse("tlh"); ok {
		t.Error("an unknown language must not parse")
	}
}

func TestDetectFollowsPOSIXOrder(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	cases := []struct {
		env  map[string]string
		want Language
	}{
		{map[string]string{"LANG": "es_AR.UTF-8"}, LangES},
		{map[string]string{"LC_ALL": "de_DE.UTF-8", "LANG": "es_AR.UTF-8"}, LangDE},
		{map[string]string{"LC_ALL": "C", "LANG": "fr_FR.UTF-8"}, LangFR},
		{map[string]string{"LANG": "tlh_XX"}, LangEN},
		{map[string]string{}, LangEN},
	}
	for _, c := range cases {
		if got := Detect(env(c.env)); got != c.want {
			t.Errorf("Detect(%v) = %v, want %v", c.env, got, c.want)
		}
	}
}

func TestTPicksTheLanguageAndFallsBackToEnglish(t *testing.T) {
	t.Cleanup(func() { SetLanguage("en") })
	SetLanguage("es")
	if got := T("Hola", "Hello"); got != "Hola" {
		t.Errorf("es = %q", got)
	}
	SetLanguage("en")
	if got := T("Hola", "Hello"); got != "Hello" {
		t.Errorf("en = %q", got)
	}
	SetLanguage("de")
	if got := T("Hola", "text without a catalog entry"); got != "text without a catalog entry" {
		t.Errorf("a missing translation must fall back to English, got %q", got)
	}
	SetLanguage("klingon")
	if CurrentLanguage() != LangEN {
		t.Errorf("an unknown language means English, got %v", CurrentLanguage())
	}
}

func TestErrorfUsesTheActiveLanguage(t *testing.T) {
	t.Cleanup(func() { SetLanguage("en") })
	SetLanguage("es")
	if got := Errorf("no puedo %s", "cannot %s", "x").Error(); got != "no puedo x" {
		t.Errorf("got %q", got)
	}
}
