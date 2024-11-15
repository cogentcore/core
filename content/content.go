// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package content provides a system for making content-focused
// apps and websites consisting of Markdown, HTML, and Cogent Core.
package content

import (
	"io/fs"

	"cogentcore.org/core/core"
)

// Content manages and displays the content of a set of pages.
type Content struct {
	core.Frame

	// Source is the source filesystem for the content.
	Source fs.FS

	// Pages are the pages that constitute the content.
	Pages []*Page `set:"-"`
}

func (ct *Content) Init() {
	ct.Frame.Init()
}
