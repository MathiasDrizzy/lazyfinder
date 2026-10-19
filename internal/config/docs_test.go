package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// docs/configuration.md documents every key of the default config, with its dotted name for nested objects
// ("preview.images"), so a new setting cannot be left out of it.
func TestTheConfigurationDocNamesEveryKey(t *testing.T) {
	doc, err := os.ReadFile("../../docs/configuration.md")
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]any
	if err := json.Unmarshal(DefaultJSON("/h"), &tree); err != nil {
		t.Fatal(err)
	}
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			key := prefix + k
			if sub, ok := v.(map[string]any); ok && k != "openers" && k != "commands" && k != "keys" && k != "icon_map" && k != "open_with" {
				walk(key+".", sub)
				continue
			}
			if !strings.Contains(string(doc), "`"+key+"`") && !inObjectRow(string(doc), prefix, k) {
				t.Errorf("docs/configuration.md lacks %q", key)
			}
		}
	}
	walk("", tree)
}

// inObjectRow says whether the row of the object prefix ("sort.") shows its key k, as in
// | `sort` | `{ "by": column, ... }` |.
func inObjectRow(doc, prefix, k string) bool {
	if prefix == "" {
		return false
	}
	for line := range strings.Lines(doc) {
		if strings.HasPrefix(line, "| `"+strings.TrimSuffix(prefix, ".")+"` |") && strings.Contains(line, `"`+k+`"`) {
			return true
		}
	}
	return false
}
