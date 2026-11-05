// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paginate

import (
	"strconv"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/printer"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

// TextStyler does standard text styling for printout:
// FontFamily and Black Color (e.g., in case user is in Dark mode).
func TextStyler(s *styles.Style) {
	s.Font.Family = printer.Settings.FontFamily
	s.Color = colors.Uniform(colors.Black)
}

// CenteredPageNumber generates a page number centered in the frame
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
	core.NewText(fr).SetText(strconv.Itoa(pageNo)).Styler(func(s *styles.Style) {
		TextStyler(s)
	})
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
			TextStyler(s)
			s.SetTextWrap(false)
			s.Font.Slant = rich.Italic
			s.Font.Size = printer.Settings.FontSize
		})
		core.NewStretch(fr)
		core.NewText(fr).SetText(strconv.Itoa(pageNo)).Styler(func(s *styles.Style) {
			TextStyler(s)
			s.SetTextWrap(false)
			s.Font.Size = printer.Settings.FontSize
		})
		core.NewSpace(frame).Styler(func(s *styles.Style) { // space after
			s.Min.Y.Em(3)
			s.Grow.Set(1, 0)
		})
	}
}

// CenteredTitle inserts centered text elements for each element if non-empty.
func CenteredTitle(title, authors, affiliations, url, date, version, reference, abstract string) func(frame *core.Frame, opts *Options) {
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
			TextStyler(s)
			s.Font.Size = printer.Settings.FontSize
			s.Font.Size.Value *= 16.0 / 11
			s.Text.Align = text.Center
		})

		if authors != "" {
			core.NewText(fr).SetText(authors).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Font.Size.Value *= 12.0 / 11
				s.Text.Align = text.Center
			})
		}

		if affiliations != "" {
			core.NewText(fr).SetText(affiliations).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.Align = text.Center
				s.Text.LineHeight = 1.1
			})
		}
		core.NewSpace(fr).Styler(func(s *styles.Style) { s.Min.Y.Em(1) })

		if date != "" {
			txt := date
			if version != "" {
				txt += ", Version: " + version
			}
			core.NewText(fr).SetText(txt).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.Align = text.Center
			})
		} else if version != "" {
			core.NewText(fr).SetText("Version: " + version).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.Align = text.Center
			})
		}

		if url != "" {
			core.NewText(fr).SetText(url).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.Align = text.Center
			})
		}

		if reference != "" {
			core.NewText(fr).SetText(reference).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.Align = text.Center
			})
		}

		if abstract != "" {
			core.NewText(fr).SetText("Abstract:").Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Font.Size.Value *= 12.0 / 11
				s.Font.Weight = rich.Bold
				s.Align.Self = styles.Start
			})
			core.NewText(fr).SetText(abstract).Styler(func(s *styles.Style) {
				TextStyler(s)
				s.Font.Size = printer.Settings.FontSize
				s.Text.LineHeight = printer.Settings.LineHeight
				s.Align.Self = styles.Start
			})
		}
		core.NewSpace(fr).Styler(func(s *styles.Style) { s.Min.Y.Em(1) })
	}
}

// APAHeaders is a TextStyler function that sets APA-style headers based
// on the tag property. The default material design header sizes used onscreen are
// generally too large for print. This is designed for content, where the
// second level header ## is used for most top-level headers within a page.
func APAHeaders(tx *core.Text) {
	headerLevel := 0
	if tag, ok := tx.Properties["tag"].(string); ok && len(tag) > 1 && tag[0] == 'h' {
		headerLevel, _ = strconv.Atoi(tag[1:])
	}
	s := &tx.Styles
	base := printer.Settings.FontSize
	// the line heights are of the types of text of the headings
	// (as h2 is [core.TextHeadlineMedium])
	switch headerLevel {
	case 1: // e.g., chapter level
		s.Text.LineHeight = 44.0 / 36
		s.Font.Size = base
		s.Font.Size.Value *= 16.0 / 11.0
		s.Font.Weight = rich.Bold
	case 2:
		s.Text.LineHeight = 36.0 / 28
		s.Font.Size = base
		s.Font.Size.Value *= 14.0 / 11.0
		s.Font.Weight = rich.Bold
	case 3:
		s.Text.LineHeight = 28.0 / 22
		s.Font.Size = base
		s.Font.Size.Value *= 12.0 / 11.0
		s.Font.Weight = rich.Bold
	case 4:
		s.Text.LineHeight = 24.0 / 16
		s.Font.Size = base
		s.Font.Weight = rich.Bold
		s.Font.Slant = rich.Italic
	case 5: // as [core.TextTitleSmall], from the 0.83em of the CSS of h5
		s.Text.LineHeight = 20.0 / 14
		s.Font.Size.Value *= (14.0 / 16) / 0.83
		s.Font.Weight = rich.Medium
	case 6: // as [core.TextLabelSmall], from the 0.67em of the CSS of h6
		s.Text.LineHeight = 16.0 / 11
		s.Font.Size.Value *= (11.0 / 16) / 0.67
		s.Font.Weight = rich.Medium
	}
}
