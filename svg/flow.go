// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

// Flow represents SVG flow* elements
type Flow struct {
	NodeBase
	FlowType string
}

func (g *Flow) SVGName() string { return "flow" }
