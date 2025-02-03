// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shaped

// Link represents a hyperlink within shaped text.
type Link struct {
	// Label is the text label for the link.
	Label string

	// URL is the full URL for the link.
	URL string

	// Properties are additional properties defined for the link,
	// e.g., from the parsed HTML attributes.
	Properties map[string]any

	// Region defines the starting and ending positions of the link,
	// in terms of shaped Lines within the containing [Lines], and Run
	// index (not character!) within each line. Links should always be
	// contained within their own separate Span in the original source.
	Region Region
}
