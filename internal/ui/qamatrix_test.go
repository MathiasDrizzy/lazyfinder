package ui

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// docs/qa-matriz.md claims a test or a scenario for every function; a name that does not exist is a lie.
func TestQAMatrixNamesExist(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "..", "docs", "qa-matriz.md"))
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	files, _ := filepath.Glob(filepath.Join("..", "*", "*_test.go"))
	cmdFiles, _ := filepath.Glob(filepath.Join("..", "..", "cmd", "*", "*_test.go")) // the command line's tests too
	files = append(files, cmdFiles...)
	for _, f := range files {
		b, _ := os.ReadFile(f)
		for _, m := range regexp.MustCompile(`(?m)^func Test(\w+)`).FindAllStringSubmatch(string(b), -1) {
			have["T "+m[1]] = true
		}
	}
	scenarios, _ := filepath.Glob(filepath.Join("..", "..", "testdata", "tuiqa", "*.json"))
	for _, s := range scenarios {
		have["S "+filepath.Base(s[:len(s)-len(".json")])] = true
	}
	rows := regexp.MustCompile(`(?m)^\|.*$`).FindAllString(string(doc), -1)
	claims := 0
	for _, row := range rows {
		for _, m := range regexp.MustCompile(`\b([TS]) ([\w-]+)`).FindAllStringSubmatch(row, -1) {
			claims++
			if !have[m[1]+" "+m[2]] {
				t.Errorf("docs/qa-matriz.md names %s %s, which does not exist", m[1], m[2])
			}
		}
		for _, m := range regexp.MustCompile(`git: Test(\w+)`).FindAllStringSubmatch(row, -1) {
			claims++
			if !have["T "+m[1]] {
				t.Errorf("docs/qa-matriz.md names git test %s, which does not exist", m[1])
			}
		}
	}
	if claims < 100 {
		t.Fatalf("only %d claims found: did the table format change?", claims)
	}
}
