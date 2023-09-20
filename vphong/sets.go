// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package vphong

// Sets are variable set numbers - must coordinate with System sets!
type Sets int //enums:enum

const (
	MtxsSet Sets = iota
	NLightSet
	LightSet
	TexSet
)
