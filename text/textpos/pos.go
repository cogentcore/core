// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import (
	"fmt"
	"strings"
)

// Pos is a text position in terms of line and character index within a line,
// using 0-based line numbers, which are converted to 1 base for the String()
// representation. Char positions are always in runes, and can also
// be used for other units such as tokens, spans, or runs.
type Pos struct {
	Line int
	Char int
}

// AddLine returns a Pos with Line number added.
func (ps Pos) AddLine(ln int) Pos {
	ps.Line += ln
	return ps
}

// AddChar returns a Pos with Char number added.
func (ps Pos) AddChar(ch int) Pos {
	ps.Char += ch
	return ps
}

// String satisfies the fmt.Stringer interferace
func (ps Pos) String() string {
	s := fmt.Sprintf("%d", ps.Line+1)
	if ps.Char != 0 {
		s += fmt.Sprintf(":%d", ps.Char)
	}
	return s
}

var (
	// PosErr represents an error text position (-1 for both line and char)
	// used as a return value for cases where error positions are possible.
	PosErr = Pos{-1, -1}

	PosZero = Pos{}
)

// IsLess returns true if receiver position is less than given comparison.
func (ps Pos) IsLess(cmp Pos) bool {
	switch {
	case ps.Line < cmp.Line:
		return true
	case ps.Line == cmp.Line:
		return ps.Char < cmp.Char
	default:
		return false
	}
}

// FromString decodes text position from a string representation of form:
// [#]LxxCxx. Used in e.g., URL links. Returns true if successful.
func (ps *Pos) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	lidx := strings.Index(link, "L")
	cidx := strings.Index(link, "C")

	switch {
	case lidx >= 0 && cidx >= 0:
		fmt.Sscanf(link, "L%dC%d", &ps.Line, &ps.Char)
		ps.Line-- // link is 1-based, we use 0-based
		ps.Char-- // ditto
	case lidx >= 0:
		fmt.Sscanf(link, "L%d", &ps.Line)
		ps.Line-- // link is 1-based, we use 0-based
	case cidx >= 0:
		fmt.Sscanf(link, "C%d", &ps.Char)
		ps.Char--
	default:
		// todo: could support other formats
		return false
	}
	return true
}
