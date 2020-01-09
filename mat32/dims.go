// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

import "github.com/goki/ki/kit"

// Dims is a list of vector dimension (component) names
type Dims int

const (
	X Dims = iota
	Y
	Z
	W
	DimsN
)

//go:generate stringer -type=Dims

var KiT_Dims = kit.Enums.AddEnum(DimsN, kit.NotBitFlag, nil)

// OtherDim returns the other dimension for 2D X,Y
func OtherDim(d Dims) Dims {
	switch d {
	case X:
		return Y
	default:
		return X
	}
}
