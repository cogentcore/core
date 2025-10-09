// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"time"

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
		if ps.SiteTitle != "" && pt == curPage.Name {
			pt = ps.SiteTitle + ": " + pt
		}
		ps.PDF.Header = paginate.NoFirst(paginate.HeaderLeftPageNumber(pt))
		ur := ct.getPrintURL() + "/" + curPage.URL
		ura := `<a href="` + ur + `">` + ur + `</a>`
		dt := ""
		if !curPage.Date.IsZero() {
			dt = curPage.Date.Format("January 2, 2006")
		} else {
			dt = time.Now().Format("January 2, 2006")
		}
		ps.PDF.Title = paginate.CenteredTitle(pt, curPage.Authors, curPage.Affiliations, ura, dt, curPage.Abstract)
		return ps
	}
}
