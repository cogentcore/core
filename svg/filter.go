// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Filter represents SVG filter* elements
type Filter struct {
	NodeBase
	FilterType string
}

var KiT_Filter = kit.Types.AddType(&Filter{}, nil)

// AddNewFilter adds a new filter to given parent node, with given name.
func AddNewFilter(parent ki.Ki, name string) *Filter {
	return parent.AddNewChild(KiT_Filter, name).(*Filter)
}
