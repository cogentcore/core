// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dirsort

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"unicode/utf16"
)

// errBplist is the error of a binary property list that is not valid.
var errBplist = errors.New("dirsort: not a valid binary property list")

// decodeBplist decodes the given binary property list (as those of the
// view settings of the Finder), whose values are nil, bool, int64,
// float64 (also for dates), []byte, string, []any and map[string]any.
func decodeBplist(b []byte) (any, error) {
	if len(b) < 8+32 || !bytes.HasPrefix(b, []byte("bplist0")) {
		return nil, errBplist
	}
	tr := b[len(b)-32:]
	d := &bplist{b: b, offSize: int(tr[6]), refSize: int(tr[7])}
	numObjs := binary.BigEndian.Uint64(tr[8:])
	top := binary.BigEndian.Uint64(tr[16:])
	tableOff := binary.BigEndian.Uint64(tr[24:])
	if d.offSize < 1 || d.offSize > 8 || d.refSize < 1 || d.refSize > 8 ||
		numObjs > uint64(len(b)) || top >= numObjs ||
		tableOff > uint64(len(b)) || numObjs*uint64(d.offSize) > uint64(len(b))-tableOff {
		return nil, errBplist
	}
	d.offsets = make([]int, numObjs)
	for i := range d.offsets {
		p := int(tableOff) + i*d.offSize
		d.offsets[i] = int(readUint(b[p : p+d.offSize]))
	}
	return d.object(int(top), 0)
}

// bplist is the state of decoding a binary property list.
type bplist struct {
	b                []byte
	offSize, refSize int
	offsets          []int
}

// readUint returns the big-endian unsigned integer of the given bytes.
func readUint(b []byte) uint64 {
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

// object decodes the object of the given index, at the given depth of
// arrays and dictionaries.
func (d *bplist) object(idx, depth int) (any, error) {
	if idx < 0 || idx >= len(d.offsets) || depth > 32 {
		return nil, errBplist
	}
	p := d.offsets[idx]
	if p < 8 || p >= len(d.b) {
		return nil, errBplist
	}
	marker := d.b[p]
	kind, low := marker>>4, int(marker&0xf)
	p++
	switch kind {
	case 0x0:
		switch marker {
		case 0x08:
			return false, nil
		case 0x09:
			return true, nil
		}
		return nil, nil
	case 0x1: // integer of 2^low bytes
		n := 1 << low
		if n > 16 || p+n > len(d.b) {
			return nil, errBplist
		}
		if n == 16 { // (the low 8 bytes of a 128-bit integer)
			return int64(readUint(d.b[p+8 : p+16])), nil
		}
		return int64(readUint(d.b[p : p+n])), nil
	case 0x2, 0x3: // real, or date (a real in seconds)
		n := 1 << low
		if kind == 0x3 {
			n = 8
		}
		if p+n > len(d.b) {
			return nil, errBplist
		}
		switch n {
		case 4:
			return float64(math.Float32frombits(uint32(readUint(d.b[p : p+4])))), nil
		case 8:
			return math.Float64frombits(readUint(d.b[p : p+8])), nil
		}
		return nil, errBplist
	}
	n, p, err := d.count(low, p)
	if err != nil {
		return nil, err
	}
	switch kind {
	case 0x4: // data
		if p+n > len(d.b) {
			return nil, errBplist
		}
		return bytes.Clone(d.b[p : p+n]), nil
	case 0x5: // ASCII string
		if p+n > len(d.b) {
			return nil, errBplist
		}
		return string(d.b[p : p+n]), nil
	case 0x6: // UTF-16 string, of n characters
		if p+2*n > len(d.b) {
			return nil, errBplist
		}
		u := make([]uint16, n)
		for i := range u {
			u[i] = binary.BigEndian.Uint16(d.b[p+2*i:])
		}
		return string(utf16.Decode(u)), nil
	case 0x8: // UID
		return int64(readUint(d.b[p : p+min(low+1, len(d.b)-p)])), nil
	case 0xa, 0xc: // array, set
		refs, err := d.refs(p, n)
		if err != nil {
			return nil, err
		}
		arr := make([]any, n)
		for i, r := range refs {
			if arr[i], err = d.object(r, depth+1); err != nil {
				return nil, err
			}
		}
		return arr, nil
	case 0xd: // dictionary: the references of the keys, then of the values
		refs, err := d.refs(p, 2*n)
		if err != nil {
			return nil, err
		}
		m := make(map[string]any, n)
		for i := range n {
			k, err := d.object(refs[i], depth+1)
			if err != nil {
				return nil, err
			}
			v, err := d.object(refs[n+i], depth+1)
			if err != nil {
				return nil, err
			}
			if ks, ok := k.(string); ok {
				m[ks] = v
			}
		}
		return m, nil
	}
	return nil, errBplist
}

// count returns the count of an object with the given low nibble of its
// marker, whose content is at the given position, which is after the
// count if it is an integer object that follows (for 15 and more), and
// that position.
func (d *bplist) count(low, p int) (int, int, error) {
	if low != 0xf {
		return low, p, nil
	}
	if p >= len(d.b) || d.b[p]>>4 != 0x1 {
		return 0, 0, errBplist
	}
	n := 1 << (d.b[p] & 0xf)
	p++
	if n > 8 || p+n > len(d.b) {
		return 0, 0, errBplist
	}
	c := readUint(d.b[p : p+n])
	if c > uint64(len(d.b)) {
		return 0, 0, errBplist
	}
	return int(c), p + n, nil
}

// refs returns the given number of object references at the given position.
func (d *bplist) refs(p, n int) ([]int, error) {
	if n < 0 || p+n*d.refSize > len(d.b) {
		return nil, errBplist
	}
	refs := make([]int, n)
	for i := range refs {
		q := p + i*d.refSize
		refs[i] = int(readUint(d.b[q : q+d.refSize]))
	}
	return refs, nil
}
