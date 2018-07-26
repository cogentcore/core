// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/ki/kit"
)

// Flow represents SVG flow* elements
type Flow struct {
	SVGNodeBase
	FlowType string
}

var KiT_Flow = kit.Types.AddType(&Flow{}, nil)
