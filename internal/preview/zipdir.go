package preview

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"io/fs"
	"strings"
)

// zipDir is what the end record of a zip says: how many entries it holds and where its central directory starts.
type zipDir struct {
	records uint64
	start   int64
}

// zipDirectory reads the end record of a zip, and its zip64 record when the end record points to one, the way
// archive/zip does, without loading the central directory. ok is false when there is no valid end record. The count
// is untrusted: archive/zip checks it only modulo 65536, so walkZip counts the records themselves.
func zipDirectory(r io.ReaderAt, size int64) (d zipDir, ok bool) {
	const eocdLen, maxComment, locatorLen, zip64Len = 22, 65535, 20, 56
	tail := min(size, eocdLen+maxComment)
	buf := make([]byte, tail)
	if _, err := r.ReadAt(buf, size-tail); err != nil && err != io.EOF {
		return zipDir{}, false
	}
	i := bytes.LastIndex(buf, []byte("PK\x05\x06"))
	if i < 0 || len(buf)-i < eocdLen {
		return zipDir{}, false
	}
	end := size - tail + int64(i)
	records := uint64(binary.LittleEndian.Uint16(buf[i+10:]))
	dirSize := uint64(binary.LittleEndian.Uint32(buf[i+12:]))
	dirOffset := uint64(binary.LittleEndian.Uint32(buf[i+16:]))
	if records == 0xffff || dirSize == 0xffffffff || dirOffset == 0xffffffff {
		var loc [locatorLen]byte // a zip64 locator sits right before the end record
		if end-locatorLen >= 0 {
			if _, err := r.ReadAt(loc[:], end-locatorLen); err != nil {
				return zipDir{}, false
			}
		}
		if string(loc[:4]) == "PK\x06\x07" {
			off := int64(binary.LittleEndian.Uint64(loc[8:]))
			var rec [zip64Len]byte
			if off < 0 || off > size-zip64Len {
				return zipDir{}, false
			}
			if _, err := r.ReadAt(rec[:], off); err != nil || string(rec[:4]) != "PK\x06\x06" {
				return zipDir{}, false
			}
			end = off
			records = binary.LittleEndian.Uint64(rec[32:])
			dirSize = binary.LittleEndian.Uint64(rec[40:])
			dirOffset = binary.LittleEndian.Uint64(rec[48:])
		}
	}
	if dirSize > 1<<62 || dirOffset > 1<<62 {
		return zipDir{}, false
	}
	// Data put before the zip (a self-extracting one) shifts every offset by base, unless the directory is already
	// where the end record says.
	base := end - int64(dirSize) - int64(dirOffset)
	if o := base + int64(dirOffset); o < 0 || o >= size {
		return zipDir{}, false
	}
	if base > 0 && zipHeaderAt(r, int64(dirOffset)) {
		base = 0
	}
	return zipDir{records: records, start: base + int64(dirOffset)}, true
}

func zipHeaderAt(r io.ReaderAt, off int64) bool {
	var sig [4]byte
	_, err := r.ReadAt(sig[:], off)
	return err == nil && string(sig[:]) == "PK\x01\x02"
}

// walkZip reads the central directory from start one record at a time and calls visit with each entry, keeping none of
// them: memory stays the same whatever the zip holds. It ends at the first bytes that are not a record, as archive/zip
// does, or when visit returns false (stopped is then true).
func walkZip(ctx context.Context, r io.ReaderAt, size, start int64, visit func(name string, size int64, isDir bool) bool) (stopped bool, err error) {
	const headerLen = 46
	br := bufio.NewReaderSize(io.NewSectionReader(r, start, size-start), 64<<10)
	var h [headerLen]byte
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if _, err := io.ReadFull(br, h[:]); err != nil || string(h[:4]) != "PK\x01\x02" {
			return false, nil
		}
		le16 := func(at int) int { return int(binary.LittleEndian.Uint16(h[at:])) }
		name := make([]byte, le16(28))
		extra := make([]byte, le16(30))
		if _, err := io.ReadFull(br, name); err != nil {
			return false, nil
		}
		if _, err := io.ReadFull(br, extra); err != nil {
			return false, nil
		}
		if _, err := br.Discard(le16(32)); err != nil {
			return false, nil
		}
		usize := uint64(binary.LittleEndian.Uint32(h[24:]))
		if usize == 0xffffffff {
			usize = zip64Size(extra, usize)
		}
		isDir := zipIsDir(string(name), h[5], binary.LittleEndian.Uint32(h[38:]))
		if !visit(string(name), int64(min(usize, 1<<62)), isDir) {
			return true, nil
		}
	}
}

// zip64Size is the uncompressed size from the zip64 extra field of a record, or def when it has none.
func zip64Size(extra []byte, def uint64) uint64 {
	for len(extra) >= 4 {
		id, n := binary.LittleEndian.Uint16(extra), int(binary.LittleEndian.Uint16(extra[2:]))
		extra = extra[4:]
		if n > len(extra) {
			break
		}
		if id == 1 && n >= 8 {
			return binary.LittleEndian.Uint64(extra)
		}
		extra = extra[n:]
	}
	return def
}

// zipIsDir says whether an entry is a folder, as zip.FileHeader.Mode does: a name ending in a slash, or the folder bit
// of the attributes of the system that made it.
func zipIsDir(name string, creator byte, attrs uint32) bool {
	const unix, macOS, ntfs, vfat, fat = 3, 19, 11, 14, 0
	switch creator {
	case unix, macOS:
		if fs.FileMode(attrs>>16)&0o170000 == 0o040000 {
			return true
		}
	case ntfs, vfat, fat:
		if attrs&0x10 != 0 {
			return true
		}
	}
	return strings.HasSuffix(name, "/")
}
