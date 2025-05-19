// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import (
	"fmt"
	"strings"
	"time"

	"cogentcore.org/core/base/nptime"
)

var RegionZero = Region{}

// Region is a contiguous region within a source file with lines of rune chars,
// defined by start and end [Pos] positions.
// End.Char position is _exclusive_ so the last char is the one before End.Char.
// End.Line position is _inclusive_, so the last line is End.Line.
// There is a Time stamp for when the region was created as valid positions
// into the lines source, which is critical for tracking edits in live documents.
type Region struct {
	// Start is the starting position of region.
	Start Pos

	// End is the ending position of region.
	// Char position is _exclusive_ so the last char is the one before End.Char.
	// Line position is _inclusive_, so the last line is End.Line.
	End Pos

	// Time when region was set: needed for updating locations in the text based
	// on time stamp (using efficient non-pointer time).
	Time nptime.Time
}

// NewRegion creates a new text region using separate line and char
// values for start and end. Sets timestamp to now.
func NewRegion(stLn, stCh, edLn, edCh int) Region {
	tr := Region{Start: Pos{Line: stLn, Char: stCh}, End: Pos{Line: edLn, Char: edCh}}
	tr.TimeNow()
	return tr
}

// NewRegionPos creates a new text region using position values.
// Sets timestamp to now.
func NewRegionPos(st, ed Pos) Region {
	tr := Region{Start: st, End: ed}
	tr.TimeNow()
	return tr
}

// NewRegionLen makes a new Region from a starting point and a length
// along same line. Sets timestamp to now.
func NewRegionLen(start Pos, len int) Region {
	tr := Region{Start: start}
	tr.End = start
	tr.End.Char += len
	tr.TimeNow()
	return tr
}

// IsNil checks if the region is empty, because the start is after or equal to the end.
func (tr Region) IsNil() bool {
	return !tr.Start.IsLess(tr.End)
}

// Contains returns true if region contains given position.
func (tr Region) Contains(ps Pos) bool {
	return ps.IsLess(tr.End) && (tr.Start == ps || tr.Start.IsLess(ps))
}

// ContainsLine returns true if line is within region
func (tr Region) ContainsLine(ln int) bool {
	return tr.Start.Line >= ln && ln <= tr.End.Line
}

// NumLines is the number of lines in this region, based on inclusive end line.
func (tr Region) NumLines() int {
	return 1 + (tr.End.Line - tr.Start.Line)
}

// Intersect returns the intersection of this region with given
// other region, where the other region is assumed to be the larger,
// constraining region, within which you are fitting the receiver region.
// Char level start / end are only constrained if on same Start / End line.
// The given endChar value is used for the end of an interior line.
func (tr Region) Intersect(or Region, endChar int) Region {
	switch {
	case tr.Start.Line < or.Start.Line:
		tr.Start = or.Start
	case tr.Start.Line == or.Start.Line:
		tr.Start.Char = max(tr.Start.Char, or.Start.Char)
	case tr.Start.Line < or.End.Line:
		tr.Start.Char = 0
	case tr.Start.Line == or.End.Line:
		tr.Start.Char = min(tr.Start.Char, or.End.Char-1)
	default:
		return Region{} // not in bounds
	}
	if tr.End.Line == tr.Start.Line { // keep valid
		tr.End.Char = max(tr.End.Char, tr.Start.Char)
	}
	switch {
	case tr.End.Line < or.End.Line:
		tr.End.Char = endChar
	case tr.End.Line == or.End.Line:
		tr.End.Char = min(tr.End.Char, or.End.Char)
	}
	return tr
}

// ShiftLines returns a new Region with the start and End lines
// shifted by given number of lines.
func (tr Region) ShiftLines(ln int) Region {
	tr.Start.Line += ln
	tr.End.Line += ln
	return tr
}

// MoveToLine returns a new Region with the Start line
// set to given line.
func (tr Region) MoveToLine(ln int) Region {
	nl := tr.NumLines()
	tr.Start.Line = 0
	tr.End.Line = nl - 1
	return tr
}

////////  Time

// TimeNow grabs the current time as the edit time.
func (tr *Region) TimeNow() {
	tr.Time.Now()
}

// IsAfterTime reports if this region's time stamp is after given time value
// if region Time stamp has not been set, it always returns true
func (tr *Region) IsAfterTime(t time.Time) bool {
	if tr.Time.IsZero() {
		return true
	}
	return tr.Time.Time().After(t)
}

// Ago returns how long ago this Region's time stamp is relative
// to given time.
func (tr *Region) Ago(t time.Time) time.Duration {
	return t.Sub(tr.Time.Time())
}

// Age returns the time interval from [time.Now]
func (tr *Region) Age() time.Duration {
	return tr.Ago(time.Now())
}

// Since returns the time interval between
// this Region's time stamp and that of the given earlier region's stamp.
func (tr *Region) Since(earlier *Region) time.Duration {
	return earlier.Ago(tr.Time.Time())
}

// FromStringURL decodes text region from a string representation of form:
// [#]LxxCxx-LxxCxx. Used in e.g., URL links. returns true if successful
func (tr *Region) FromStringURL(link string) bool {
	link = strings.TrimPrefix(link, "#")
	fmt.Sscanf(link, "L%dC%d-L%dC%d", &tr.Start.Line, &tr.Start.Char, &tr.End.Line, &tr.End.Char)
	return true
}

func (tr *Region) String() string {
	return fmt.Sprintf("[%s - %s]", tr.Start, tr.End)
}
