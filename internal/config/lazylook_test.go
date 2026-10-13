package config

import (
	"slices"
	"strings"
	"testing"
)

func TestDefaultsOfIconSizeAndColumnSeparators(t *testing.T) {
	cfg := Defaults("/home/u")
	if cfg.IconSize != "medium" || !cfg.ColumnSeparators {
		t.Fatalf("defaults: icon_size %q, column_separators %v", cfg.IconSize, cfg.ColumnSeparators)
	}
}

func TestParseIconSizeAndColumnSeparators(t *testing.T) {
	cfg, err := Parse([]byte(`{"version":1,"icon_size":"large","column_separators":false}`), "/h")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IconSize != "large" || cfg.ColumnSeparators {
		t.Fatalf("parsed wrong: %q %v", cfg.IconSize, cfg.ColumnSeparators)
	}
	if _, err := Parse([]byte(`{"version":1,"icon_size":"huge"}`), "/h"); err == nil || !strings.Contains(err.Error(), `"icon_size"`) {
		t.Fatalf("want an icon_size error, got %v", err)
	}
}

func TestOwnerAndPathAreColumns(t *testing.T) {
	if _, err := Parse([]byte(`{"version":1,"columns":["name","owner","path"]}`), "/h"); err != nil {
		t.Fatal(err)
	}
}

func TestColumnPresets(t *testing.T) {
	basic, full := ColumnPresets["basic"], ColumnPresets["full"]
	if !slices.Equal(basic, []string{"name", "modified", "size", "kind"}) {
		t.Fatalf("basic = %v", basic)
	}
	if len(full) != len(validColumns) || full[0] != "name" {
		t.Fatalf("full must list every column, name first: %v", full)
	}
	if ColumnPreset(basic) != "basic" || ColumnPreset(full) != "full" || ColumnPreset([]string{"name", "size"}) != "custom" {
		t.Fatal("ColumnPreset names the preset a list matches")
	}
}
