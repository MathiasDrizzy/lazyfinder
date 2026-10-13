package i18n

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// sourceKeys walks the repo (tests excluded) and returns every translatable text, English -> Spanish: the two literal
// arguments of each i18n.T call. An argument that is not a fixed text is returned in dynamic, so nobody adds a text
// that escapes the coverage test.
func sourceKeys(t *testing.T) (keys map[string]string, dynamic []string) {
	keys, dynamic, _ = sourceTexts(t)
	return keys, dynamic
}

// sourceTexts is sourceKeys plus every string literal of the code: the texts that i18n.E and i18n.Errorf translate
// (error messages, the kinds of file) are looked up by their English literal, so a catalog entry for one of them is
// not stale.
func sourceTexts(t *testing.T) (keys map[string]string, dynamic []string, literals map[string]bool) {
	t.Helper()
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	var files []*ast.File
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			f, perr := parser.ParseFile(fset, p, nil, 0)
			if perr != nil {
				t.Fatal(perr)
			}
			files = append(files, f)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	lit := func(e ast.Expr) (string, bool) {
		bl, ok := e.(*ast.BasicLit)
		if !ok || bl.Kind != token.STRING {
			return "", false
		}
		s, err := strconv.Unquote(bl.Value)
		return s, err == nil
	}
	keys = map[string]string{}
	literals = map[string]bool{}
	for _, f := range files {
		if f.Name.Name == "i18n" {
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := lit0(n); ok {
				literals[s] = true
			}
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 2 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				return true
			}
			if id, ok := sel.X.(*ast.Ident); !ok || id.Name != "i18n" {
				return true
			}
			es, ok1 := lit(call.Args[0])
			en, ok2 := lit(call.Args[1])
			if !ok1 || !ok2 {
				dynamic = append(dynamic, fset.Position(call.Pos()).String())
				return true
			}
			if prev, dup := keys[en]; dup && prev != es {
				t.Errorf("the English text %q has two Spanish versions: %q and %q", en, prev, es)
			}
			keys[en] = es
			return true
		})
	}
	return keys, dynamic, literals
}

func lit0(n ast.Node) (string, bool) {
	bl, ok := n.(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(bl.Value)
	return s, err == nil
}

func TestEveryTextIsAFixedPair(t *testing.T) {
	_, dynamic := sourceKeys(t)
	if len(dynamic) > 0 {
		t.Fatalf("i18n.T needs two string literals; these calls do not have them:\n%s", strings.Join(dynamic, "\n"))
	}
}

func TestSpanishIsNeverEmpty(t *testing.T) {
	keys, _ := sourceKeys(t)
	for en, es := range keys {
		if strings.TrimSpace(es) == "" || strings.TrimSpace(en) == "" {
			t.Errorf("empty text in the pair %q / %q", es, en)
		}
	}
}

func verbs(s string) string {
	var out []string
	for i := 0; i < len(s)-1; i++ {
		if s[i] == '%' {
			if s[i+1] == '%' {
				i++
				continue
			}
			out = append(out, string(s[i+1]))
		}
	}
	return strings.Join(out, "")
}

func TestCatalogsCoverEveryTextAndNothingElse(t *testing.T) {
	keys, _, literals := sourceTexts(t)
	for _, lang := range Languages {
		if lang == LangEN || lang == LangES {
			continue
		}
		cat := catalog(lang)
		for en := range keys {
			tr, ok := cat[en]
			if !ok || strings.TrimSpace(tr) == "" {
				t.Errorf("%s: missing translation of %q", lang, en)
				continue
			}
			if verbs(tr) != verbs(en) {
				t.Errorf("%s: %q and its translation %q use different format verbs", lang, en, tr)
			}
		}
		for en := range cat {
			if _, ok := keys[en]; !ok && !literals[en] {
				t.Errorf("%s: %q is in the catalog but not in the code (stale)", lang, en)
			}
		}
	}
}

// plainInLazymark are texts that lazymark never translates: a value such as terminal reads the same everywhere.
var plainInLazymark = []string{"terminal"}

// TestStringsShared compares, key by key, the texts lazyfinder has in common with lazymark (a committed snapshot,
// see scripts/snapshot-lazymark-strings.py): the same English text must have the same Spanish and the same
// translation in every language, so the two apps sound alike.
func TestStringsShared(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "lazymark.json"))
	if err != nil {
		t.Fatal(err)
	}
	var snap struct {
		Source  string
		Es      map[string]string
		Catalog map[string]map[string]string
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	keys, _ := sourceKeys(t)
	shared := 0
	for en, es := range keys {
		if want, ok := snap.Es[en]; ok {
			shared++
			if es != want {
				t.Errorf("%q: lazyfinder says %q, %s says %q", en, es, snap.Source, want)
			}
			for lang, cat := range snap.Catalog {
				if got, mine := catalog(Language(lang))[en], cat[en]; mine != "" && got != mine {
					t.Errorf("%s: %q is %q here and %q in %s", lang, en, got, mine, snap.Source)
				}
			}
		}
	}
	for _, en := range plainInLazymark {
		if es, ok := keys[en]; ok {
			shared++
			if es != en {
				t.Errorf("%q: lazymark leaves it as is, lazyfinder says %q", en, es)
			}
			for _, lang := range Languages {
				if lang != LangEN && lang != LangES && catalog(lang)[en] != en {
					t.Errorf("%s: %q should stay %q, is %q", lang, en, en, catalog(lang)[en])
				}
			}
		}
	}
	if shared < 5 {
		t.Fatalf("only %d texts shared with %s: the snapshot or the sweep is broken", shared, snap.Source)
	}
}
