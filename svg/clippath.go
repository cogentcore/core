// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// todo: needs to be impl

// ClipPath is used for holding a path that renders as a clip path
type ClipPath struct {
	NodeBase
}

var KiT_ClipPath = kit.Types.AddType(&ClipPath{}, nil)

// AddNewClipPath adds a new clippath to given parent node, with given name.
func AddNewClipPath(parent ki.Ki, name string) *ClipPath {
	return parent.AddNewChild(KiT_ClipPath, name).(*ClipPath)
}
