// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bcontent

// Site contains the data printed in JSON by a content app run with
// the -generatehtml argument, which is then used by the core build tool
// to make the website for the content, consisting of static HTML pages
// for all of the content pages. These pages are a preview that is shown
// while the Go wasm app is loading, or a freestanding website without
// the wasm app at all (htmlonly mode).
type Site struct {

	// Pages are the pre-rendered pages, in the order they are listed
	// in the content source.
	Pages []*PreRenderPage

	// Categories are the page categories, in the order they are displayed
	// in the categories tree (most pages first, with "Other" last).
	Categories []*Category

	// CSS is additional CSS needed for the pre-rendered HTML,
	// such as the styles for code syntax highlighting.
	CSS string
}

// Category is one category of pages in a [Site].
type Category struct {

	// Name is the name of the category.
	Name string

	// HasPage is whether there is a page with the same name as the
	// category, which is opened when the category is selected.
	HasPage bool

	// URL is the URL of the category page if [Category.HasPage].
	URL string

	// Pages are the pages in the category, excluding the category page itself.
	Pages []*NavPage
}

// NavPage is the information about a page needed for navigation.
type NavPage struct {
	Name string
	URL  string
}

// TOCHeading is one heading in the table of contents for a page.
type TOCHeading struct {

	// Level is the heading level (1-6).
	Level int

	// Text is the text of the heading, which may contain HTML formatting.
	Text string

	// ID is the HTML id of the heading element.
	ID string
}
