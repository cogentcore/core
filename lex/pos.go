// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lex

import (
	"fmt"
	"strings"

	"goki.dev/pi/v2/token"
)

// Pos is a position within the source file -- it is recorded always in 0, 0
// offset positions, but is converted into 1,1 offset for public consumption
// Ch positions are always in runes, not bytes.  Also used for lex token indexes.
type Pos struct {
	Ln int
	Ch int
}

// String satisfies the fmt.Stringer interferace
func (ps Pos) String() string {
	s := fmt.Sprintf("%d", ps.Ln+1)
	if ps.Ch != 0 {
		s += fmt.Sprintf(":%d", ps.Ch)
	}
	return s
}

// PosZero is the uninitialized zero text position (which is
// still a valid position)
var PosZero = Pos{}

// PosErr represents an error text position (-1 for both line and char)
// used as a return value for cases where error positions are possible
var PosErr = Pos{-1, -1}

// IsLess returns true if receiver position is less than given comparison
func (ps *Pos) IsLess(cmp Pos) bool {
	switch {
	case ps.Ln < cmp.Ln:
		return true
	case ps.Ln == cmp.Ln:
		return ps.Ch < cmp.Ch
	default:
		return false
	}
}

// FromString decodes text position from a string representation of form:
// [#]LxxCxx -- used in e.g., URL links -- returns true if successful
func (ps *Pos) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	lidx := strings.Index(link, "L")
	cidx := strings.Index(link, "C")

	switch {
	case lidx >= 0 && cidx >= 0:
		fmt.Sscanf(link, "L%dC%d", &ps.Ln, &ps.Ch)
		ps.Ln-- // link is 1-based, we use 0-based
		ps.Ch-- // ditto
	case lidx >= 0:
		fmt.Sscanf(link, "L%d", &ps.Ln)
		ps.Ln-- // link is 1-based, we use 0-based
	case cidx >= 0:
		fmt.Sscanf(link, "C%d", &ps.Ch)
		ps.Ch--
	default:
		// todo: could support other formats
		return false
	}
	return true
}

////////////////////////////////////////////////////////////////////
//  Reg

// Reg is a contiguous region within the source file
type Reg struct {

	// starting position of region
	St Pos `desc:"starting position of region"`

	// ending position of region
	Ed Pos `desc:"ending position of region"`
}

// RegZero is the zero region
var RegZero = Reg{}

// IsNil checks if the region is empty, because the start is after or equal to the end
func (tr Reg) IsNil() bool {
	return !tr.St.IsLess(tr.Ed)
}

// Contains returns true if region contains position
func (tr Reg) Contains(ps Pos) bool {
	return ps.IsLess(tr.Ed) && (tr.St == ps || tr.St.IsLess(ps))
}

////////////////////////////////////////////////////////////////////
//  EosPos

// EosPos is a line of EOS token positions, always sorted low-to-high
type EosPos []int

// FindGt returns any pos value greater than given token pos, -1 if none
func (ep EosPos) FindGt(ch int) int {
	for i := range ep {
		if ep[i] > ch {
			return ep[i]
		}
	}
	return -1
}

// FindGtEq returns any pos value greater than or equal to given token pos, -1 if none
func (ep EosPos) FindGtEq(ch int) int {
	for i := range ep {
		if ep[i] >= ch {
			return ep[i]
		}
	}
	return -1
}

////////////////////////////////////////////////////////////////////
//  TokenMap

// TokenMap is a token map, for optimizing token exclusion
type TokenMap map[token.Tokens]struct{}

// Set sets map for given token
func (tm TokenMap) Set(tok token.Tokens) {
	tm[tok] = struct{}{}
}

// Has returns true if given token is in the map
func (tm TokenMap) Has(tok token.Tokens) bool {
	_, has := tm[tok]
	return has
}
