// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package badatomic has an atomic bit flag type that is not int32 or int64,
// which enumgen must reject.
package badatomic

type Sizes int16 //enums:bitflag -atomic

const (
	Small Sizes = iota
	Large
)
