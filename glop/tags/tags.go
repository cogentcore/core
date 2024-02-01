// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tags provides support for advanced struct tags defined
// using a TypeTags method.
package tags

// TypeTagger is the interface that types can implement to define
// advanced struct tags in code.
type TypeTagger interface {

	// TypeTags returns the advanced struct tags for the fields
	// of this type.
	TypeTags() map[string]map[string]any
}
