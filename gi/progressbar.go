// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"sync"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/mat32/v2"
)

// ProgressBar is a progress bar that fills up bar as progress continues.
// Call Start with a maximum value to work toward, and ProgStep each time
// a progress step has been accomplished -- increments the ProgCur by one
// and display is updated every ProgInc such steps.
type ProgressBar struct {
	Slider

	// maximum amount of progress to be achieved
	ProgMax int

	// progress increment when display is updated -- automatically computed from ProgMax at Start but can be overwritten
	ProgInc int

	// current progress level
	ProgCur int

	// mutex for updating progress
	ProgMu sync.Mutex `set:"-"`
}

func (pb *ProgressBar) CopyFieldsFrom(frm any) {
	fr := frm.(*ProgressBar)
	pb.Slider.CopyFieldsFrom(&fr.Slider)
}

func (pb *ProgressBar) OnInit() {
	pb.Type = SliderScrollbar
	pb.Dim = mat32.X
	pb.ThumbSize.Set(1, 1)
	pb.Value = 0
	pb.Step = 0.1
	pb.PageStep = 0.2
	pb.Max = 1.0
	pb.Prec = 9
	pb.SetReadOnly(true)

	pb.HandleWidgetEvents()
	pb.ProgressBarStyles()
}

func (pb *ProgressBar) ProgressBarStyles() {
	pb.Style(func(s *styles.Style) {
		pb.ValueColor.SetSolid(colors.Scheme.Primary.Base)
		pb.ThumbColor.SetSolid(colors.Scheme.Primary.Base)

		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerHighest)

		s.Color = colors.Scheme.OnSurface

		s.Padding.Zero()

		if pb.Dim == mat32.X {
			s.Min.X.Em(20)
			s.Min.Y.Dp(10)
		} else {
			s.Min.Y.Em(20)
			s.Min.X.Dp(10)
		}
	})
}

func ProgressDefaultInc(max int) int {
	switch {
	case max > 50000:
		return 1000
	case max > 5000:
		return 100
	case max > 500:
		return 10
	}
	return 1
}

func (pb *ProgressBar) Start(mx int) {
	pb.ProgMax = mx - 1
	pb.ProgMax = max(1, pb.ProgMax)
	pb.ProgInc = ProgressDefaultInc(mx)
	pb.ProgCur = 0
	pb.UpdtBar()
}

func (pb *ProgressBar) UpdtBar() {
	updt := pb.UpdateStart()
	pb.SetVisiblePct(float32(pb.ProgCur) / float32(pb.ProgMax))
	pb.UpdateEndRender(updt)
}

// ProgStep is called every time there is an increment of progress.
// This is threadsafe to call from different routines.
func (pb *ProgressBar) ProgStep() {
	pb.ProgMu.Lock()
	pb.ProgCur++
	if pb.ProgCur%pb.ProgInc == 0 {
		pb.UpdtBar()
	}
	pb.ProgMu.Unlock()
}
