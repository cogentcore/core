// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode/utf16"
)

// errDSStore is the error of a .DS_Store file that is not valid.
var errDSStore = errors.New("dirsort: not a valid .DS_Store file")

// dsStoreRecords returns the records of the given file name (as "." for
// the directory itself) in the given .DS_Store file of the Finder, by
// their codes (as "lsvP" for the settings of the list view), whose values
// are []byte (blob), string (ustr, and the four characters of type), bool,
// or int64 (long, shor, comp, dutc).
//
// A .DS_Store file is a "buddy allocator" file of blocks, one of which is
// the "DSDB" B-tree of the records of the files, sorted by file name.
func dsStoreRecords(b []byte, name string) (map[string]any, error) {
	ds := &dsStore{b: b}
	if len(b) < 36 || binary.BigEndian.Uint32(b) != 1 || !bytes.Equal(b[4:8], []byte("Bud1")) {
		return nil, errDSStore
	}
	// the offsets are of the data after the first 4 bytes
	rootOff := int(binary.BigEndian.Uint32(b[8:]))
	root := ds.reader(rootOff+4, int(binary.BigEndian.Uint32(b[12:])))
	nblocks := int(root.u32())
	root.u32()
	if nblocks <= 0 || nblocks > len(b) {
		return nil, errDSStore
	}
	ds.blocks = make([]uint32, nblocks)
	for i := range ds.blocks {
		ds.blocks[i] = root.u32()
	}
	root.skip(4 * ((256 - nblocks%256) % 256)) // (padded to 256 addresses)
	dsdb := -1
	for range int(root.u32()) {
		n := int(root.u8())
		key := root.bytes(n)
		v := int(root.u32())
		if string(key) == "DSDB" {
			dsdb = v
		}
	}
	if root.err != nil || dsdb < 0 {
		return nil, errDSStore
	}
	hdr := ds.block(dsdb)
	node := int(hdr.u32())
	if hdr.err != nil {
		return nil, errDSStore
	}
	recs := map[string]any{}
	if err := ds.node(node, name, recs, 0); err != nil {
		return nil, err
	}
	return recs, nil
}

// dsStore is the state of reading a .DS_Store file.
type dsStore struct {
	b []byte

	// blocks are the addresses of the blocks: the offset of each, with
	// its size as a power of 2 in the low 5 bits.
	blocks []uint32
}

// block returns a reader of the block of the given number.
func (ds *dsStore) block(n int) *dsReader {
	if n < 0 || n >= len(ds.blocks) {
		return &dsReader{err: errDSStore}
	}
	addr := ds.blocks[n]
	return ds.reader(int(addr&^0x1f)+4, 1<<(addr&0x1f))
}

// reader returns a reader of the given size of the file at the given offset.
func (ds *dsStore) reader(off, size int) *dsReader {
	if off < 0 || size < 0 || off > len(ds.b) || size > len(ds.b)-off {
		return &dsReader{err: errDSStore}
	}
	return &dsReader{b: ds.b[off : off+size]}
}

// node adds the records of the given file name in the node of the B-tree
// in the given block, and its children, to the given records.
func (ds *dsStore) node(n int, name string, recs map[string]any, depth int) error {
	if depth > 32 {
		return errDSStore
	}
	r := ds.block(n)
	right := int(r.u32()) // the last child of an internal node, else 0
	count := int(r.u32())
	for range count {
		if r.err != nil {
			return r.err
		}
		if right != 0 {
			if err := ds.node(int(r.u32()), name, recs, depth+1); err != nil {
				return err
			}
		}
		fname, code, v := r.record()
		if r.err == nil && fname == name {
			recs[code] = v
		}
	}
	if r.err != nil {
		return r.err
	}
	if right != 0 {
		return ds.node(right, name, recs, depth+1)
	}
	return nil
}

// dsReader reads big-endian values from a block of a .DS_Store file,
// with the first error that it has.
type dsReader struct {
	b   []byte
	p   int
	err error
}

// bytes returns the next n bytes.
func (r *dsReader) bytes(n int) []byte {
	if r.err != nil || n < 0 || r.p+n > len(r.b) {
		r.err = errDSStore
		return nil
	}
	v := r.b[r.p : r.p+n]
	r.p += n
	return v
}

// skip skips the next n bytes.
func (r *dsReader) skip(n int) { r.bytes(n) }

// u8 returns the next byte.
func (r *dsReader) u8() uint8 {
	if b := r.bytes(1); b != nil {
		return b[0]
	}
	return 0
}

// u32 returns the next 32-bit integer.
func (r *dsReader) u32() uint32 {
	if b := r.bytes(4); b != nil {
		return binary.BigEndian.Uint32(b)
	}
	return 0
}

// utf16 returns the next UTF-16 string of n characters.
func (r *dsReader) utf16(n int) string {
	b := r.bytes(2 * n)
	if b == nil {
		return ""
	}
	u := make([]uint16, n)
	for i := range u {
		u[i] = binary.BigEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(u))
}

// record returns the next record: the name of its file, its code, and its
// value.
func (r *dsReader) record() (name, code string, v any) {
	name = r.utf16(int(r.u32()))
	code = string(r.bytes(4))
	switch string(r.bytes(4)) {
	case "bool":
		v = r.u8() != 0
	case "long", "shor":
		v = int64(int32(r.u32()))
	case "type":
		v = string(r.bytes(4))
	case "comp", "dutc":
		v = int64(uint64(r.u32())<<32 | uint64(r.u32()))
	case "blob":
		v = bytes.Clone(r.bytes(int(r.u32())))
	case "ustr":
		v = r.utf16(int(r.u32()))
	default:
		r.err = errDSStore
	}
	return
}
