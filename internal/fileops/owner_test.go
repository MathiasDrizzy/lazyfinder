package fileops

import (
	"os/user"
	"strconv"
	"testing"
)

// Get Info shows who owns an item, by name when the system knows it and by number otherwise.
func TestInfoNamesTheOwner(t *testing.T) {
	w := newWorld(t)
	in, err := Info(w.path("src/a.txt"))
	w.must(err)
	me, err := user.Current()
	if err != nil {
		t.Skip("no current user:", err)
	}
	if in.Owner != me.Username && in.Owner != me.Uid {
		t.Fatalf("owner = %q, want %q or %q", in.Owner, me.Username, me.Uid)
	}
	if in.Owner == "" || ownerName(123456789) != strconv.Itoa(123456789) {
		t.Fatalf("an unknown id must show as its number: %q", ownerName(123456789))
	}
}
