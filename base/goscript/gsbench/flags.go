// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gsbench

//go:generate core generate

// ladFlags are bit flags like those of text/rich.Decorations, for the
// LadderBitFlag benchmarks.
type ladFlags int32 //enums:bitflag

const (
	ladFlagA ladFlags = iota
	ladFlagB
	ladFlagC
	ladFlagD
)

// ladFlagged is a struct with a bit flag field, like rich.Style.
type ladFlagged struct {
	Flags ladFlags
	Size  int
}
