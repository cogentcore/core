// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"fmt"
	"strings"
	"time"

	"github.com/goki/ki/nptime"
	"github.com/goki/pi/lex"
)

// Region represents a text region as a start / end position, and includes
// a Time stamp for when the region was created as valid positions into the TextBuf.
// The character end position is an *exclusive* position (i.e., the region ends at
// the character just prior to that character) but the lines are always *inclusive*
// (i.e., it is the actual line, not the next line).
type Region struct {
	Start lex.Pos     `desc:"starting position"`
	End   lex.Pos     `desc:"ending position: line number is *inclusive* but character position is *exclusive* (-1)"`
	Time  nptime.Time `desc:"time when region was set -- needed for updating locations in the text based on time stamp (using efficient non-pointer time)"`
}

// RegionNil is the empty (zero) text region -- all zeros
var RegionNil Region

// IsNil checks if the region is empty, because the start is after or equal to the end
func (tr *Region) IsNil() bool {
	return !tr.Start.IsLess(tr.End)
}

// IsSameLine returns true if region starts and ends on the same line
func (tr *Region) IsSameLine() bool {
	return tr.Start.Ln == tr.End.Ln
}

// Contains returns true if line is within region
func (tr *Region) Contains(ln int) bool {
	if tr.Start.Ln >= ln && ln <= tr.End.Ln {
		return true
	}
	return false
}

// TimeNow grabs the current time as the edit time
func (tr *Region) TimeNow() {
	tr.Time.Now()
}

// NewRegion creates a new text region using separate line and char
// values for start and end, and also sets the time stamp to now
func NewRegion(stLn, stCh, edLn, edCh int) Region {
	tr := Region{Start: lex.Pos{Ln: stLn, Ch: stCh}, End: lex.Pos{Ln: edLn, Ch: edCh}}
	tr.TimeNow()
	return tr
}

// NewRegionPos creates a new text region using position values
// and also sets the time stamp to now
func NewRegionPos(st, ed lex.Pos) Region {
	tr := Region{Start: st, End: ed}
	tr.TimeNow()
	return tr
}

// IsAfterTime reports if this region's time stamp is after given time value
// if region Time stamp has not been set, it always returns true
func (tr *Region) IsAfterTime(t time.Time) bool {
	if tr.Time.IsZero() {
		return true
	}
	return tr.Time.Time().After(t)
}

// AgoMSec returns how long ago this Region's time stamp is relative
// to given time, in milliseconds.
func (tr *Region) AgoMSec(t time.Time) int {
	return int(t.Sub(tr.Time.Time()) / time.Millisecond)
}

// AgeMSec returns the time interval in milliseconds from time.Now()
func (tr *Region) AgeMSec() int {
	return tr.AgoMSec(time.Now())
}

// SinceMSec returns the time interval in milliseconds between
// this Region's time stamp and that of the given earlier region's stamp.
func (tr *Region) SinceMSec(earlier *Region) int {
	return earlier.AgoMSec(tr.Time.Time())
}

// FromString decodes text region from a string representation of form:
// [#]LxxCxx-LxxCxx -- used in e.g., URL links -- returns true if successful
func (tr *Region) FromString(link string) bool {
	link = strings.TrimPrefix(link, "#")
	fmt.Sscanf(link, "L%dC%d-L%dC%d", &tr.Start.Ln, &tr.Start.Ch, &tr.End.Ln, &tr.End.Ch)
	tr.Start.Ln--
	tr.Start.Ch--
	tr.End.Ln--
	tr.End.Ch--
	return true
}

// NewRegionLen makes a new Region from a starting point and a length
// along same line
func NewRegionLen(start lex.Pos, len int) Region {
	reg := Region{}
	reg.Start = start
	reg.End = start
	reg.End.Ch += len
	return reg
}
