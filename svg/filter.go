// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

// Filter represents SVG filter* elements
type Filter struct {
	NodeBase
	FilterType string
}

func (g *Filter) SVGName() string { return "filter" }
