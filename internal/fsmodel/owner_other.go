//go:build !unix

package fsmodel

import "os"

// ownerOf is empty where the file mode carries no owner.
func ownerOf(os.FileInfo) string { return "" }
