// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

// Types is an enum with all
// of the possible types of packages.
type Types int //enums:enum

const (
	// TypeApp is an executable app
	TypeApp Types = iota
	// TypeLibrary is an importable library
	TypeLibrary
)
