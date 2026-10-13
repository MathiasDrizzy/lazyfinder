package preview

import (
	"fmt"
	"strings"
)

// DefaultHexBytes is how much of a binary file the hex view shows.
const DefaultHexBytes = 4096

// hexRows formats b like `hexdump -C`: offset, two groups of eight bytes and the printable characters.
func hexRows(b []byte) []string {
	rows := make([]string, 0, (len(b)+15)/16)
	for off := 0; off < len(b); off += 16 {
		end := min(off+16, len(b))
		var hex, text strings.Builder
		for i := off; i < off+16; i++ {
			if i == off+8 {
				hex.WriteByte(' ')
			}
			if i < end {
				fmt.Fprintf(&hex, "%02x ", b[i])
				text.WriteByte(printable(b[i]))
			} else {
				hex.WriteString("   ")
			}
		}
		rows = append(rows, fmt.Sprintf("%08x  %s |%s|", off, hex.String(), text.String()))
	}
	return rows
}

func printable(c byte) byte {
	if c >= 0x20 && c < 0x7f {
		return c
	}
	return '.'
}
