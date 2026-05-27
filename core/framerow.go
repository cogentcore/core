package core

import (
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
)

func NewFrameRow(parent ...tree.Node) *Frame {
	frm := NewFrame(parent...)
	frm.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // no scrollbars!
		s.Gap.Set(units.Dp(4))
		s.Align.Items = styles.Center
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Wrap = true
	})
	return frm
}
