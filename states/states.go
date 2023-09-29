// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package states

//go:generate enumgen

// States are GUI states that are relevant for styling
type States int64 //enums:bitflag

const (
	Disabled States = iota
	ReadOnly
	Selected
	Active
	Focused
	FocusWithin
	Checked
	Hovered
	Pressed
	Invalid
	Link
)
