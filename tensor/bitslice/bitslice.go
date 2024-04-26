// Copyright (c) 2024, The Cogent Core Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bitslice implements a simple slice-of-bits using a []byte slice for storage,
// which is used for efficient storage of boolean data, such as projection connectivity patterns.
package bitslice

import "fmt"

// bitslice.Slice is the slice of []byte that holds the bits.
// first byte maintains the number of bits used in the last byte (0-7).
// when 0 then prior byte is all full and a new one must be added for append.
type Slice []byte

// BitIndex returns the byte, bit index of given bit index
func BitIndex(idx int) (byte int, bit uint32) {
	return idx / 8, uint32(idx % 8)
}

// Make makes a new bitslice of given length and capacity (optional, pass 0 for default)
// *bits* (rounds up 1 for both).
// also reserves first byte for extra bits value
func Make(ln, cp int) Slice {
	by, bi := BitIndex(ln)
	bln := by
	if bi != 0 {
		bln++
	}
	var sl Slice
	if cp > 0 {
		sl = make(Slice, bln+1, (cp/8)+2)
	} else {
		sl = make(Slice, bln+1)
	}
	sl[0] = byte(bi)
	return sl
}

// Len returns the length of the slice in bits
func (bs *Slice) Len() int {
	ln := len(*bs)
	if ln == 0 {
		return 0
	}
	eb := (*bs)[0]
	bln := ln - 1
	if eb != 0 {
		bln--
	}
	tln := bln*8 + int(eb)
	return tln
}

// Cap returns the capacity of the slice in bits -- always modulo 8
func (bs *Slice) Cap() int {
	return (cap(*bs) - 1) * 8
}

// SetLen sets the length of the slice, copying values if a new allocation is required
func (bs *Slice) SetLen(ln int) {
	by, bi := BitIndex(ln)
	bln := by
	if bi != 0 {
		bln++
	}
	if cap(*bs) >= bln+1 {
		*bs = (*bs)[0 : bln+1]
		(*bs)[0] = byte(bi)
	} else {
		sl := make(Slice, bln+1)
		sl[0] = byte(bi)
		copy(sl, *bs)
		*bs = sl
	}
}

// Set sets value of given bit index -- no extra range checking is performed -- will panic if out of range
func (bs *Slice) Set(idx int, val bool) {
	by, bi := BitIndex(idx)
	if val {
		(*bs)[by+1] |= 1 << bi
	} else {
		(*bs)[by+1] &^= 1 << bi
	}
}

// Index returns bit value at given bit index
func (bs *Slice) Index(idx int) bool {
	by, bi := BitIndex(idx)
	return ((*bs)[by+1] & (1 << bi)) != 0
}

// Append adds a bit to the slice and returns possibly new slice, possibly old slice..
func (bs *Slice) Append(val bool) Slice {
	if len(*bs) == 0 {
		*bs = Make(1, 0)
		bs.Set(0, val)
		return *bs
	}
	ln := bs.Len()
	eb := (*bs)[0]
	if eb == 0 {
		*bs = append(*bs, 0) // now we add
		(*bs)[0] = 1
	} else if eb < 7 {
		(*bs)[0]++
	} else {
		(*bs)[0] = 0
	}
	bs.Set(ln, val)
	return *bs
}

// SetAll sets all values to either on or off -- much faster than setting individual bits
func (bs *Slice) SetAll(val bool) {
	ln := len(*bs)
	for i := 1; i < ln; i++ {
		if val {
			(*bs)[i] = 0xFF
		} else {
			(*bs)[i] = 0
		}
	}
}

// ToBools converts to a []bool slice
func (bs *Slice) ToBools() []bool {
	ln := len(*bs)
	bb := make([]bool, ln)
	for i := 0; i < ln; i++ {
		bb[i] = bs.Index(i)
	}
	return bb
}

// Clone creates a new copy of this bitslice with separate memory
func (bs *Slice) Clone() Slice {
	cp := make(Slice, len(*bs))
	copy(cp, *bs)
	return cp
}

// SubSlice returns a new Slice from given start, end range indexes of this slice
// if end is <= 0 then the length of the source slice is used (equivalent to omitting
// the number after the : in a Go subslice expression)
func (bs *Slice) SubSlice(start, end int) Slice {
	ln := bs.Len()
	if end <= 0 {
		end = ln
	}
	if end > ln {
		panic("bitslice.SubSlice: end index is beyond length of slice")
	}
	if start > end {
		panic("bitslice.SubSlice: start index greater than end index")
	}
	nln := end - start
	if nln <= 0 {
		return Slice{}
	}
	ss := Make(nln, 0)
	for i := 0; i < nln; i++ {
		ss.Set(i, bs.Index(i+start))
	}
	return ss
}

// Delete returns a new bit slice with N elements removed starting at given index.
// This must be a copy given the nature of the 8-bit aliasing.
func (bs *Slice) Delete(start, n int) Slice {
	ln := bs.Len()
	if n <= 0 {
		panic("bitslice.Delete: n <= 0")
	}
	if start >= ln {
		panic("bitslice.Delete: start index >= length")
	}
	end := start + n
	if end > ln {
		panic("bitslice.Delete: end index greater than length")
	}
	nln := ln - n
	if nln <= 0 {
		return Slice{}
	}
	ss := Make(nln, 0)
	for i := 0; i < start; i++ {
		ss.Set(i, bs.Index(i))
	}
	for i := end; i < ln; i++ {
		ss.Set(i-n, bs.Index(i))
	}
	return ss
}

// Insert returns a new bit slice with N false elements inserted starting at given index.
// This must be a copy given the nature of the 8-bit aliasing.
func (bs *Slice) Insert(start, n int) Slice {
	ln := bs.Len()
	if n <= 0 {
		panic("bitslice.Insert: n <= 0")
	}
	if start > ln {
		panic("bitslice.Insert: start index greater than length")
	}
	nln := ln + n
	ss := Make(nln, 0)
	for i := 0; i < start; i++ {
		ss.Set(i, bs.Index(i))
	}
	for i := start; i < ln; i++ {
		ss.Set(i+n, bs.Index(i))
	}
	return ss
}

// String satisfies the fmt.Stringer interface
func (bs *Slice) String() string {
	ln := bs.Len()
	if ln == 0 {
		if *bs == nil {
			return "nil"
		}
		return "[]"
	}
	mx := ln
	if mx > 1000 {
		mx = 1000
	}
	str := "["
	for i := 0; i < mx; i++ {
		val := bs.Index(i)
		if val {
			str += "1 "
		} else {
			str += "0 "
		}
		if (i+1)%80 == 0 {
			str += "\n"
		}
	}
	if ln > mx {
		str += fmt.Sprintf("...(len=%v)", ln)
	}
	str += "]"
	return str
}
