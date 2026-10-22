// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package badbits has a bit flag type with more flags than bits,
// which enumgen must reject.
package badbits

type Bits int8 //enums:bitflag

const (
	B0 Bits = iota
	B1
	B2
	B3
	B4
	B5
	B6
	B7
	B8
)
