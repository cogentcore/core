// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// SizeClasses are the different size classes that a window can have
type SizeClasses int32 //enums:enum

const (
	// Compact is the size class for windows with a width less than
	// 600dp, which typically happens on phones.
	Compact SizeClasses = iota

	// Medium is the size class for windows with a width between 600dp
	// and 840dp inclusive, which typically happens on tablets.
	Medium

	// Expanded is the size class for windows with a width greater than
	// 840dp, which typically happens on desktop and laptop computers.
	Expanded
)

// SizeClass returns the size class of the scene in which it is contained.
func (wb *WidgetBase) SizeClass() SizeClasses {
	dots := float32(wb.Sc.SceneGeom.Size.X)
	dpd := wb.Sc.Styles.UnContext.Dp(1) // dots per dp
	dp := dots / dpd                    // dots / (dots / dp) = dots * (dp / dots) = dp
	switch {
	case dp < 600:
		return Compact
	case dp > 840:
		return Expanded
	default:
		return Medium
	}
}
