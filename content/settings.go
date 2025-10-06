// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"strings"

	"cogentcore.org/core/content/bcontent"
	"cogentcore.org/core/text/paginate"
	"cogentcore.org/core/text/rich"
)

func init() {
	Settings.Defaults()
}

// Settings are the current settings for content rendering.
var Settings SettingsData

// SettingsData has settings parameters for content,
// including PDF rendering options.
type SettingsData struct {
	PDF paginate.Options

	// SiteTitle is the title of the site, used in page headings and titles.
	SiteTitle string

	// PageSettings is a function that returns the settings data to use
	// for the current page. Can set custom parameters for different pages.
	// The default sets the PDF Header function to HeaderLeftPageNumber
	// with current page Title.
	PageSettings func(ct *Content, curPage *bcontent.Page) *SettingsData
}

func (s *SettingsData) Defaults() {
	s.PDF.Defaults()
	s.PDF.FontFamily = rich.Serif
	s.PDF.Footer = nil

	s.PageSettings = func(ct *Content, curPage *bcontent.Page) *SettingsData {
		ps := &SettingsData{}
		*ps = Settings
		pt := curPage.Title
		if ps.SiteTitle != "" {
			pt = ps.SiteTitle + ": " + pt
		}
		ps.PDF.Header = paginate.HeaderLeftPageNumber(pt)
		au := ""
		if len(curPage.Authors) > 0 {
			au = strings.Join(curPage.Authors, ", ")
		}
		// todo: add affiliations and abstract
		af := ct.getPrintURL() + "/" + curPage.URL
		ps.PDF.Title = paginate.CenteredTitle(pt, au, af, "")
		return ps
	}
}
