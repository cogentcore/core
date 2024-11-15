// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import "time"

// Page represents the metadata for a single page of content.
type Page struct {

	// Filename is the name of the file that the content is stored in.
	Filename string `toml:"-"`

	// Name is the user-friendly name of the page, defaulting to the
	// [strcase.ToSentence] of the [Page.Filename].
	Name string

	// Date is the optional date that the page was published.
	Date time.Time

	// Authors are the optional authors of the page.
	Authors []string

	// Draft indicates that the page is a draft and should not be visible on the web.
	Draft bool

	// Categories are the categories that the page belongs to.
	Categories []string
}
