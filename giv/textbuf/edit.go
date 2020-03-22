// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textbuf

import (
	"time"

	"github.com/goki/ki/sliceclone"
	"github.com/goki/pi/lex"
)

// Edit describes an edit action to a buffer -- this is the data passed
// via signals to viewers of the buffer.  Actions are only deletions and
// insertions (a change is a sequence of those, given normal editing
// processes).  The TextBuf always reflects the current state *after* the edit.
type Edit struct {
	Reg    Region   `desc:"region for the edit (start is same for previous and current, end is in original pre-delete text for a delete, and in new lines data for an insert.  Also contains the Time stamp for this edit."`
	Text   [][]rune `desc:"text deleted or inserted -- in lines.  For Rect this is just for the spanning character distance per line, times number of lines."`
	Group  int      `desc:"optional grouping number, for grouping edits in Undo for example"`
	Delete bool     `desc:"action is either a deletion or an insertion"`
	Rect   bool     `desc:"this is a rectangular region with upper left corner = Reg.Start and lower right corner = Reg.End -- otherwise it is for the full continuous region."`
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

// AdjustPosDel determines what to do with positions within deleted region
type AdjustPosDel int

// these are options for what to do with positions within deleted region
// for the AdjustPos function
const (
	// AdjustPosDelErr means return a PosErr when in deleted region
	AdjustPosDelErr AdjustPosDel = iota

	// AdjustPosDelStart means return start of deleted region
	AdjustPosDelStart

	// AdjustPosDelEnd means return end of deleted region
	AdjustPosDelEnd
)

// Clone returns a clone of the edit record.
func (te *Edit) Clone() *Edit {
	rc := &Edit{}
	rc.CopyFrom(te)
	return rc
}

// Copy copies from other Edit
func (te *Edit) CopyFrom(cp *Edit) {
	te.Reg = cp.Reg
	te.Group = cp.Group
	te.Delete = cp.Delete
	te.Rect = cp.Rect
	nln := len(cp.Text)
	if nln == 0 {
		te.Text = nil
	}
	te.Text = make([][]rune, nln)
	for i, r := range cp.Text {
		te.Text[i] = sliceclone.Rune(r)
	}
}

// AdjustPos adjusts the given text position as a function of the edit.
// if the position was within a deleted region of text, del determines
// what is returned
func (te *Edit) AdjustPos(pos lex.Pos, del AdjustPosDel) lex.Pos {
	if te == nil {
		return pos
	}
	if pos.IsLess(te.Reg.Start) || pos == te.Reg.Start {
		return pos
	}
	dl := te.Reg.End.Ln - te.Reg.Start.Ln
	if pos.Ln > te.Reg.End.Ln {
		if te.Delete {
			pos.Ln -= dl
		} else {
			pos.Ln += dl
		}
		return pos
	}
	if te.Delete {
		if pos.Ln < te.Reg.End.Ln || pos.Ch < te.Reg.End.Ch {
			switch del {
			case AdjustPosDelStart:
				return te.Reg.Start
			case AdjustPosDelEnd:
				return te.Reg.End
			case AdjustPosDelErr:
				return lex.PosErr
			}
		}
		// this means pos.Ln == te.Reg.End.Ln, Ch >= end
		if dl == 0 {
			pos.Ch -= (te.Reg.End.Ch - te.Reg.Start.Ch)
		} else {
			pos.Ch -= te.Reg.End.Ch
		}
	} else {
		if dl == 0 {
			pos.Ch += (te.Reg.End.Ch - te.Reg.Start.Ch)
		} else {
			pos.Ln += dl
		}
	}
	return pos
}

// AdjustPosIfAfterTime checks the time stamp and IfAfterTime,
// it adjusts the given text position as a function of the edit
// del determines what to do with positions within a deleted region
// either move to start or end of the region, or return an error.
func (te *Edit) AdjustPosIfAfterTime(pos lex.Pos, t time.Time, del AdjustPosDel) lex.Pos {
	if te == nil {
		return pos
	}
	if te.Reg.IsAfterTime(t) {
		return te.AdjustPos(pos, del)
	}
	return pos
}

// AdjustReg adjusts the given text region as a function of the edit, including
// checking that the timestamp on the region is after the edit time, if
// the region has a valid Time stamp (otherwise always does adjustment).
// If the starting position is within a deleted region, it is moved to the
// end of the deleted region, and if the ending position was within a deleted
// region, it is moved to the start.  If the region becomes empty, RegionNil
// will be returned.
func (te *Edit) AdjustReg(reg Region) Region {
	if te == nil {
		return reg
	}
	if !reg.Time.IsZero() && !te.Reg.IsAfterTime(reg.Time.Time()) {
		return reg
	}
	reg.Start = te.AdjustPos(reg.Start, AdjustPosDelEnd)
	reg.End = te.AdjustPos(reg.End, AdjustPosDelStart)
	if reg.IsNil() {
		return RegionNil
	}
	return reg
}
