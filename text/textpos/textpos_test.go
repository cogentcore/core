// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPos(t *testing.T) {
	p := Pos{Line: 2, Char: 5}
	assert.Equal(t, Pos{4, 5}, p.AddLine(2))
	assert.Equal(t, Pos{2, 2}, p.AddChar(-3))
	assert.Equal(t, Pos{2, 5}, p, "the receiver is unchanged")
	assert.Equal(t, "3:5", p.String(), "1-based line")
	assert.Equal(t, "1", Pos{}.String(), "no char when it is 0")

	tests := []struct {
		a, b Pos
		less bool
	}{
		{Pos{0, 5}, Pos{1, 0}, true},
		{Pos{1, 0}, Pos{0, 5}, false},
		{Pos{1, 2}, Pos{1, 3}, true},
		{Pos{1, 3}, Pos{1, 3}, false},
		{Pos{1, 4}, Pos{1, 3}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.less, tt.a.IsLess(tt.b), "%v < %v", tt.a, tt.b)
	}
}

func TestPosFromString(t *testing.T) {
	tests := []struct {
		s    string
		want Pos
		ok   bool
	}{
		{"L3C5", Pos{2, 4}, true},
		{"#L3C5", Pos{2, 4}, true},
		{"L10", Pos{9, 0}, true},
		{"C7", Pos{0, 6}, true},
		{"x", Pos{}, false},
	}
	for _, tt := range tests {
		var p Pos
		ok := p.FromString(tt.s)
		assert.Equal(t, tt.ok, ok, tt.s)
		assert.Equal(t, tt.want, p, tt.s)
	}
}

func TestRange(t *testing.T) {
	r := Range{Start: 2, End: 6}
	assert.Equal(t, 4, r.Len())
	assert.True(t, r.Contains(2))
	assert.True(t, r.Contains(5))
	assert.False(t, r.Contains(6), "the end is exclusive")
	assert.False(t, r.Contains(1))
	assert.Equal(t, Range{4, 6}, r.Intersect(Range{4, 9}))
	assert.Equal(t, Range{2, 3}, r.Intersect(Range{0, 3}))
	assert.Equal(t, Range{3, 5}, r.Intersect(Range{3, 5}))
	assert.Equal(t, Range{-1, -1}, r.Intersect(Range{6, 9}), "no overlap")
}

func TestRegionConstructors(t *testing.T) {
	r := NewRegion(1, 2, 3, 4)
	assert.Equal(t, Pos{1, 2}, r.Start)
	assert.Equal(t, Pos{3, 4}, r.End)
	assert.False(t, r.Time.IsZero())
	assert.Equal(t, r.Start, NewRegionPos(Pos{1, 2}, Pos{3, 4}).Start)
	rl := NewRegionLen(Pos{1, 2}, 5)
	assert.Equal(t, Pos{1, 7}, rl.End)
	assert.Equal(t, "[2:2 - 4:4]", r.String())
}

func TestRegion(t *testing.T) {
	r := NewRegion(1, 2, 3, 4)
	assert.False(t, r.IsNil())
	assert.True(t, NewRegion(1, 2, 1, 2).IsNil())
	assert.True(t, NewRegion(2, 0, 1, 5).IsNil())

	assert.True(t, r.Contains(Pos{1, 2}), "the start is included")
	assert.True(t, r.Contains(Pos{2, 100}))
	assert.False(t, r.Contains(Pos{3, 4}), "the end is excluded")
	assert.False(t, r.Contains(Pos{1, 1}))

	assert.Equal(t, 3, r.NumLines())
	sh := r.ShiftLines(2)
	assert.Equal(t, Pos{3, 2}, sh.Start)
	assert.Equal(t, Pos{5, 4}, sh.End)

	var u Region
	assert.True(t, u.FromStringURL("#L1C2-L3C4"))
	assert.Equal(t, Pos{1, 2}, u.Start, "note: not converted from 1-based, unlike Pos.FromString")
	assert.Equal(t, Pos{3, 4}, u.End)
}

// TestRegionLines tests ContainsLine and MoveToLine, which are wrong:
// see the skip messages.
func TestRegionLines(t *testing.T) {
	t.Run("ContainsLine", func(t *testing.T) {
		r := NewRegion(2, 0, 5, 0)
		assert.True(t, r.ContainsLine(3))
		assert.False(t, r.ContainsLine(0))
		assert.False(t, r.ContainsLine(6))
	})
	t.Run("MoveToLine", func(t *testing.T) {
		r := NewRegion(2, 1, 4, 3).MoveToLine(7)
		assert.Equal(t, Pos{7, 1}, r.Start)
		assert.Equal(t, Pos{9, 3}, r.End)
	})
}

func TestRegionIntersect(t *testing.T) {
	bounds := NewRegion(2, 3, 6, 8)
	tests := []struct {
		name string
		r    Region
		want Region
	}{
		{"starts before", NewRegion(0, 5, 3, 4), Region{Start: Pos{2, 3}, End: Pos{3, 80}}},
		{"same start line", NewRegion(2, 1, 2, 2), Region{Start: Pos{2, 3}, End: Pos{2, 80}}}, // a line before the end line ends at endChar
		{"interior", NewRegion(4, 5, 6, 20), Region{Start: Pos{4, 0}, End: Pos{6, 8}}},
		{"on the end line", NewRegion(6, 10, 6, 12), Region{Start: Pos{6, 7}, End: Pos{6, 8}}},
		{"after", NewRegion(7, 0, 8, 0), Region{}},
	}
	for _, tt := range tests {
		got := tt.r.Intersect(bounds, 80)
		assert.Equal(t, tt.want.Start, got.Start, tt.name)
		assert.Equal(t, tt.want.End, got.End, tt.name)
	}
}

func TestRegionTime(t *testing.T) {
	var r Region
	assert.True(t, r.IsAfterTime(time.Now()), "a zero time is after any")
	r.TimeNow()
	before := time.Now().Add(-time.Hour)
	assert.True(t, r.IsAfterTime(before))
	assert.False(t, r.IsAfterTime(time.Now().Add(time.Hour)))
	assert.GreaterOrEqual(t, r.Age(), time.Duration(0))
	assert.InDelta(t, float64(time.Hour), float64(r.Ago(r.Time.Time().Add(time.Hour))), 1)

	earlier := Region{}
	earlier.Time.SetTime(r.Time.Time().Add(-time.Minute))
	assert.Equal(t, time.Minute, r.Since(&earlier))
}

func TestNewMatch(t *testing.T) {
	MatchContext = 3
	defer func() { MatchContext = 30 }()
	rn := []rune("hello wide world")
	m := NewMatch(rn, 6, 10, 2)
	assert.Equal(t, NewRegion(2, 6, 2, 10).Start, m.Region.Start)
	assert.Equal(t, Pos{2, 10}, m.Region.End)
	assert.Equal(t, "lo wide wo", string(m.Text), "with context")
	assert.Equal(t, Range{3, 7}, m.TextMatch)
	assert.Equal(t, "wide", string(m.Text[m.TextMatch.Start:m.TextMatch.End]))
	assert.Equal(t, "[3:6 - 3:10]: lo wide wo", m.String())

	m = NewMatch(rn, 0, 2, 0)
	assert.Equal(t, "hello", string(m.Text), "context is clipped at the start")
	assert.Equal(t, Range{0, 2}, m.TextMatch)
}

func TestEditFromRunes(t *testing.T) {
	ed := NewEditFromRunes([]rune("ab\ncde"))
	assert.Equal(t, Pos{0, 0}, ed.Region.Start)
	assert.Equal(t, Pos{1, 3}, ed.Region.End)
	assert.Equal(t, "ab\ncde", string(ed.ToBytes()))
	assert.Equal(t, "x", string(NewEditFromRunes([]rune("x")).ToBytes()))

	empty := NewEditFromRunes(nil)
	assert.Nil(t, empty.ToBytes())
	var nilEdit *Edit
	assert.Nil(t, nilEdit.ToBytes())
}

func TestEditCloneString(t *testing.T) {
	ed := NewEditFromRunes([]rune("ab\ncd"))
	ed.Delete = true
	ed.Rect = true
	ed.Group = 3
	cl := ed.Clone()
	assert.Equal(t, ed.Text, cl.Text)
	cl.Text[0][0] = 'z'
	assert.Equal(t, 'a', ed.Text[0][0], "the clone is deep")
	s := ed.String()
	assert.Contains(t, s, "[Rect]")
	assert.Contains(t, s, "[Delete]")
	assert.Contains(t, s, "Gp: 3")
	assert.Contains(t, s, "1\tcd")

	var e2 Edit
	e2.Copy(&Edit{Group: 1})
	assert.Nil(t, e2.Text)
	assert.Equal(t, 1, e2.Group)
}

func TestEditAdjustPos(t *testing.T) {
	// an insertion of 3 chars at line 1, char 2
	ins := &Edit{Region: NewRegion(1, 2, 1, 5)}
	// a deletion of chars 2 to 5 of line 1
	del := &Edit{Region: NewRegion(1, 2, 1, 5), Delete: true}
	// an insertion of 2 lines (ending at char 1 of line 3) at line 1, char 2
	insLines := &Edit{Region: NewRegion(1, 2, 3, 1)}
	// a deletion of lines 1 char 2 to 3 char 1
	delLines := &Edit{Region: NewRegion(1, 2, 3, 1), Delete: true}

	tests := []struct {
		name string
		ed   *Edit
		pos  Pos
		del  AdjustPosDel
		want Pos
	}{
		{"before insert", ins, Pos{1, 1}, AdjustPosDelErr, Pos{1, 1}},
		{"at insert", ins, Pos{1, 2}, AdjustPosDelErr, Pos{1, 2}},
		{"after insert", ins, Pos{1, 4}, AdjustPosDelErr, Pos{1, 7}},
		{"line after insert", ins, Pos{2, 4}, AdjustPosDelErr, Pos{2, 4}},
		{"in delete err", del, Pos{1, 3}, AdjustPosDelErr, PosErr},
		{"in delete start", del, Pos{1, 3}, AdjustPosDelStart, Pos{1, 2}},
		{"in delete end", del, Pos{1, 3}, AdjustPosDelEnd, Pos{1, 5}},
		{"after delete", del, Pos{1, 7}, AdjustPosDelErr, Pos{1, 4}},
		{"lines after line insert", insLines, Pos{5, 3}, AdjustPosDelErr, Pos{7, 3}},
		{"lines after line delete", delLines, Pos{5, 3}, AdjustPosDelErr, Pos{3, 3}},
		{"nil edit", nil, Pos{1, 1}, AdjustPosDelErr, Pos{1, 1}},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, tt.ed.AdjustPos(tt.pos, tt.del), tt.name)
	}
}

// TestEditAdjustPosMultiLine tests adjusting a position on the line of a
// multi-line edit, which is wrong: see the skip message.
func TestEditAdjustPosMultiLine(t *testing.T) {
	// "abcdef" with "x\ny" inserted at char 2 is "abx", "ycdef":
	// the f at char 5 is at line 2, char 4.
	ins := &Edit{Region: NewRegion(1, 2, 2, 1)}
	assert.Equal(t, Pos{2, 4}, ins.AdjustPos(Pos{1, 5}, AdjustPosDelErr))
	// deleting that again moves the f back to line 1, char 5.
	del := &Edit{Region: NewRegion(1, 2, 2, 1), Delete: true}
	assert.Equal(t, Pos{1, 5}, del.AdjustPos(Pos{2, 4}, AdjustPosDelErr))
}

func TestEditAdjustRegion(t *testing.T) {
	ins := &Edit{Region: NewRegion(0, 0, 0, 2)}
	reg := Region{Start: Pos{0, 3}, End: Pos{0, 5}}
	got := ins.AdjustRegion(reg)
	assert.Equal(t, Pos{0, 5}, got.Start)
	assert.Equal(t, Pos{0, 7}, got.End)

	reg.Time.SetTime(time.Now().Add(time.Hour))
	assert.Equal(t, reg, ins.AdjustRegion(reg), "an edit before the region is not applied")

	del := &Edit{Region: NewRegion(0, 0, 0, 10), Delete: true}
	assert.Equal(t, Region{}, del.AdjustRegion(Region{Start: Pos{0, 3}, End: Pos{0, 5}}), "a deleted region is nil")

	var nilEdit *Edit
	assert.Equal(t, reg, nilEdit.AdjustRegion(reg))

	assert.Equal(t, Pos{0, 7}, ins.AdjustPosIfAfterTime(Pos{0, 5}, time.Now().Add(-time.Hour), AdjustPosDelErr))
	assert.Equal(t, Pos{0, 5}, ins.AdjustPosIfAfterTime(Pos{0, 5}, time.Now().Add(time.Hour), AdjustPosDelErr))
	assert.Equal(t, Pos{0, 5}, nilEdit.AdjustPosIfAfterTime(Pos{0, 5}, time.Now(), AdjustPosDelErr))
}
