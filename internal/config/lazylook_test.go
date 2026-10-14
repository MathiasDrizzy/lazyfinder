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

// Mathias, 2026-10-14: "en configuraciones en columnas debería aparecer fecha de creación, extensión sobra porque
// aparece por ejemplo readme.md en su nombre". Basic is the default columns, created included; full is every column
// but the extension, which the name already shows (it can still be added from the header menu).
func TestColumnPresets(t *testing.T) {
	basic, full := ColumnPresets["basic"], ColumnPresets["full"]
	if !slices.Equal(basic, Defaults("/home/u").Columns) || !slices.Contains(basic, "created") {
		t.Fatalf("basic must be the default columns, created included: %v", basic)
	}
	if slices.Contains(full, "extension") {
		t.Fatalf("full must not show the extension, the name shows it: %v", full)
	}
	if len(full) != len(validColumns)-1 || full[0] != "name" {
		t.Fatalf("full must list every other column, name first: %v", full)
	}
	if !validColumns["extension"] {
		t.Fatal("a saved config with the extension column stays valid")
	}
	if ColumnPreset(basic) != "basic" || ColumnPreset(full) != "full" || ColumnPreset([]string{"name", "size"}) != "custom" {
		t.Fatal("ColumnPreset names the preset a list matches")
	}
}
