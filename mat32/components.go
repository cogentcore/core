// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import "github.com/goki/ki/kit"

// Components is a list of vector component names
type Components int

const (
	X Components = iota
	Y
	Z
	W
	ComponentsN
)

//go:generate stringer -type=Components

var KiT_Components = kit.Enums.AddEnum(ComponentsN, false, nil)
