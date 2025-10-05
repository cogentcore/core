// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
)

// CenteredPageNumber generates a page number cenetered in the frame
// with a 1.5em space above it.
func CenteredPageNumber(frame *core.Frame, opts *Options, pageNo int) {
	core.NewSpace(frame).Styler(func(s *styles.Style) { // space before
		s.Min.Y.Em(1.5)
		s.Grow.Set(1, 0)
	})
	fr := core.NewFrame(frame)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Justify.Content = styles.Center
	})
	core.NewText(fr).SetText(strconv.Itoa(pageNo))
}

// CenteredPageNumberNoFirst generates a page number cenetered in the frame
// with a 1.5em space above it. Skips the first one.
func CenteredPageNumberNoFirst(frame *core.Frame, opts *Options, pageNo int) {
	if pageNo == 1 {
		return
	}
	core.NewSpace(frame).Styler(func(s *styles.Style) { // space before
		s.Min.Y.Em(1.5)
		s.Grow.Set(1, 0)
	})
	fr := core.NewFrame(frame)
	fr.Styler(func(s *styles.Style) {
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Justify.Content = styles.Center
	})
	core.NewText(fr).SetText(strconv.Itoa(pageNo))
}

// HeaderLeftPageNumber adds a running header with page number on the right.
func HeaderLeftPageNumber(header string) func(frame *core.Frame, opts *Options, pageNo int) {
	return func(frame *core.Frame, opts *Options, pageNo int) {
		core.NewStretch(frame)
		fr := core.NewFrame(frame)
		fr.Styler(func(s *styles.Style) {
			s.Direction = styles.Row
			s.Grow.Set(1, 0)
		})
		core.NewText(fr).SetText(header).Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Font.Family = opts.FontFamily
			s.Font.Slant = rich.Italic
			s.Font.Size.Pt(11)
		})
		core.NewStretch(fr)
		core.NewText(fr).SetText(strconv.Itoa(pageNo)).Styler(func(s *styles.Style) {
			s.SetTextWrap(false)
			s.Font.Family = opts.FontFamily
			s.Font.Size.Pt(11)
		})
		core.NewSpace(frame).Styler(func(s *styles.Style) { // space after
			s.Min.Y.Em(3)
			s.Grow.Set(1, 0)
		})
	}
}
