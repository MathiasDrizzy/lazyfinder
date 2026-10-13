package config

import (
	"slices"
	"testing"
)

func TestDefaultOfColumnSeparators(t *testing.T) {
	if !Defaults("/home/u").ColumnSeparators {
		t.Fatal("the column separators are on by default")
	}
}

// icon_size is gone (docs/adr/0015-no-icon-size.md): a file from 0.10 or 0.11 still loads, whatever it says.
func TestAnOldIconSizeStillLoadsAndColumnSeparatorsParse(t *testing.T) {
	for _, v := range []string{"small", "medium", "large", "huge"} {
		cfg, err := Parse([]byte(`{"version":1,"icon_size":"`+v+`","column_separators":false}`), "/h")
		if err != nil {
			t.Fatalf("icon_size %q: %v", v, err)
		}
		if cfg.ColumnSeparators {
			t.Fatal("column_separators: false was not read")
		}
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
