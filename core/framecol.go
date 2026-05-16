package core

import (
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

func NewFrameCol(parent ...tree.Node) *Frame {
	frm := NewFrame(parent...)
	frm.Styler(func(s *styles.Style) {
		//s.GrowWrap = false
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Grow.X = 1
		s.Grow.Y = 1
		s.Border.Radius = styles.BorderRadiusFull
		s.Gap.Zero()
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Overflow.Set(styles.OverflowAuto)
	})
	return frm
}

