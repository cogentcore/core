// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mat32

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

// OtherDim returns the other dimension for 2D X,Y
func OtherDim(d Dims) Dims {
	switch d {
	case X:
		return Y
	default:
		return X
	}
}
