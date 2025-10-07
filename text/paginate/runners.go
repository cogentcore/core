// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"strconv"

	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
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

// NoFirst excludes the first page for any runner
func NoFirst(fun func(frame *core.Frame, opts *Options, pageNo int)) func(frame *core.Frame, opts *Options, pageNo int) {
	return func(frame *core.Frame, opts *Options, pageNo int) {
		if pageNo == 1 {
			return
		}
		fun(frame, opts, pageNo)
	}
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

// CenteredTitle inserts centered text elements for each element if non-empty.
func CenteredTitle(title, authors, affiliations, url, date, abstract string) func(frame *core.Frame, opts *Options) {
	return func(frame *core.Frame, opts *Options) {
		fr := core.NewFrame(frame)
		fr.Styler(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Grow.Set(1, 0)
			s.Align.Items = styles.Center
		})
		fr.SetProperty("paginate-block", true)

		core.NewStretch(fr).Styler(func(s *styles.Style) { // need this to take up the full width
			s.Grow.Set(1, 0)
			s.Min.X.Dot(opts.BodyDots.X)
			s.Min.Y.Em(.1)
		})
		core.NewText(fr).SetText(title).Styler(func(s *styles.Style) {
			s.Font.Family = opts.FontFamily
			s.Font.Size.Pt(16)
			s.Text.Align = text.Center
		})

		if authors != "" {
			core.NewText(fr).SetText(authors).Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(11)
				s.Text.Align = text.Center
			})
		}

		if affiliations != "" {
			core.NewText(fr).SetText(affiliations).Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(10)
				s.Text.Align = text.Center
				s.Text.LineHeight = 1.1
			})
		}
		core.NewSpace(fr).Styler(func(s *styles.Style) { s.Min.Y.Em(1) })

		if date != "" {
			core.NewText(fr).SetText(date).Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(10)
				s.Text.Align = text.Center
			})
		}

		if url != "" {
			core.NewText(fr).SetText(url).Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(10)
				s.Text.Align = text.Center
			})
		}

		if abstract != "" {
			core.NewText(fr).SetText("Abstract:").Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(11)
				s.Font.Weight = rich.Bold
				s.Align.Self = styles.Start
			})
			core.NewText(fr).SetText(abstract).Styler(func(s *styles.Style) {
				s.Font.Family = opts.FontFamily
				s.Font.Size.Pt(10)
				s.Align.Self = styles.Start
			})
		}
		core.NewSpace(fr).Styler(func(s *styles.Style) { s.Min.Y.Em(1) })
	}
}
