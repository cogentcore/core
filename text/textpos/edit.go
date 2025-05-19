// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

//go:generate core generate

import (
	"fmt"
	"slices"
	"time"

	"cogentcore.org/core/text/runes"
)

// Edit describes an edit action to line-based text, operating on
// a [Region] of the text.
// Actions are only deletions and insertions (a change is a sequence
// of each, given normal editing processes).
type Edit struct {

	// Region for the edit, specifying the region to delete, or the size
	// of the region to insert, corresponding to the Text.
	// Also contains the Time stamp for this edit.
	Region Region

	// Text deleted or inserted, in rune lines. For Rect this is the
	// spanning character distance per line, times number of lines.
	Text [][]rune

	// Group is the optional grouping number, for grouping edits in Undo for example.
	Group int

	// Delete indicates a deletion, otherwise an insertion.
	Delete bool

	// Rect is a rectangular region with upper left corner = Region.Start
	// and lower right corner = Region.End.
	// Otherwise it is for the full continuous region.
	Rect bool
}

// NewEditFromRunes returns a 0-based edit from given runes.
func NewEditFromRunes(text []rune) *Edit {
	if len(text) == 0 {
		return &Edit{}
	}
	lns := runes.Split(text, []rune("\n"))
	nl := len(lns)
	ec := len(lns[nl-1])
	ed := &Edit{}
	ed.Region = NewRegion(0, 0, nl-1, ec)
	ed.Text = lns
	return ed
}

// ToBytes returns the Text of this edit record to a byte string, with
// newlines at end of each line -- nil if Text is empty
func (te *Edit) ToBytes() []byte {
	if te == nil {
		return nil
	}
	sz := len(te.Text)
	if sz == 0 {
		return nil
	}
	if sz == 1 {
		return []byte(string(te.Text[0]))
	}
	tsz := 0
	for i := range te.Text {
		tsz += len(te.Text[i]) + 10 // don't bother converting to runes, just extra slack
	}
	b := make([]byte, 0, tsz)
	for i := range te.Text {
		b = append(b, []byte(string(te.Text[i]))...)
		if i < sz-1 {
			b = append(b, '\n')
		}
	}
	return b
}

// AdjustPos adjusts the given text position as a function of the edit.
// If the position was within a deleted region of text, del determines
// what is returned.
func (te *Edit) AdjustPos(pos Pos, del AdjustPosDel) Pos {
	if te == nil {
		return pos
	}
	if pos.IsLess(te.Region.Start) || pos == te.Region.Start {
		return pos
	}
	dl := te.Region.End.Line - te.Region.Start.Line
	if pos.Line > te.Region.End.Line {
		if te.Delete {
			pos.Line -= dl
		} else {
			pos.Line += dl
		}
		return pos
	}
	if te.Delete {
		if pos.Line < te.Region.End.Line || pos.Char < te.Region.End.Char {
			switch del {
			case AdjustPosDelStart:
				return te.Region.Start
			case AdjustPosDelEnd:
				return te.Region.End
			case AdjustPosDelErr:
				return PosErr
			}
		}
		// this means pos.Line == te.Region.End.Line, Ch >= end
		if dl == 0 {
			pos.Char -= (te.Region.End.Char - te.Region.Start.Char)
		} else {
			pos.Char -= te.Region.End.Char
		}
	} else {
		if dl == 0 {
			pos.Char += (te.Region.End.Char - te.Region.Start.Char)
		} else {
			pos.Line += dl
		}
	}
	return pos
}

// AdjustPosDel determines what to do with positions within deleted region
type AdjustPosDel int32 //enums:enum

// these are options for what to do with positions within deleted region
// for the AdjustPos function
const (
	// AdjustPosDelErr means return a PosErr when in deleted region.
	AdjustPosDelErr AdjustPosDel = iota

	// AdjustPosDelStart means return start of deleted region.
	AdjustPosDelStart

	// AdjustPosDelEnd means return end of deleted region.
	AdjustPosDelEnd
)

// Clone returns a clone of the edit record.
func (te *Edit) Clone() *Edit {
	rc := &Edit{}
	rc.Copy(te)
	return rc
}

// Copy copies from other Edit, making a clone of the source text.
func (te *Edit) Copy(cp *Edit) {
	*te = *cp
	nl := len(cp.Text)
	if nl == 0 {
		te.Text = nil
		return
	}
	te.Text = make([][]rune, nl)
	for i, r := range cp.Text {
		te.Text[i] = slices.Clone(r)
	}
}

// AdjustPosIfAfterTime checks the time stamp and IfAfterTime,
// it adjusts the given text position as a function of the edit
// del determines what to do with positions within a deleted region
// either move to start or end of the region, or return an error.
func (te *Edit) AdjustPosIfAfterTime(pos Pos, t time.Time, del AdjustPosDel) Pos {
	if te == nil {
		return pos
	}
	if te.Region.IsAfterTime(t) {
		return te.AdjustPos(pos, del)
	}
	return pos
}

// AdjustRegion adjusts the given text region as a function of the edit, including
// checking that the timestamp on the region is after the edit time, if
// the region has a valid Time stamp (otherwise always does adjustment).
// If the starting position is within a deleted region, it is moved to the
// end of the deleted region, and if the ending position was within a deleted
// region, it is moved to the start.
func (te *Edit) AdjustRegion(reg Region) Region {
	if te == nil {
		return reg
	}
	if !reg.Time.IsZero() && !te.Region.IsAfterTime(reg.Time.Time()) {
		return reg
	}
	reg.Start = te.AdjustPos(reg.Start, AdjustPosDelEnd)
	reg.End = te.AdjustPos(reg.End, AdjustPosDelStart)
	if reg.IsNil() {
		return Region{}
	}
	return reg
}

func (te *Edit) String() string {
	str := te.Region.String()
	if te.Rect {
		str += " [Rect]"
	}
	if te.Delete {
		str += " [Delete]"
	}
	str += fmt.Sprintf(" Gp: %d\n", te.Group)
	for li := range te.Text {
		str += fmt.Sprintf("%d\t%s\n", li, string(te.Text[li]))
	}
	return str
}
