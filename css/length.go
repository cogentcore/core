// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gogi

import (
// "github.com/go-gl/mathgl/mgl32"
)

// CSSLength is the <length> type in css
type CSSLength int32

const (
	LenPct  CSSLength = iota // percentage of surrounding contextual element (e.g., ViewBox)
	LenEm   CSSLength = iota // font size of the element
	LenEx   CSSLength = iota // x-height of the element's font
	LenCh   CSSLength = iota // with of the '0' glyph in the element's font
	LenRem  CSSLength = iota // font size of the root element
	LenVw   CSSLength = iota // 1% of the viewport's width
	LenVh   CSSLength = iota // 1% of the viewport's height
	LenVmin CSSLength = iota // 1% of the viewport's smaller dimension
	LenVmax CSSLength = iota // 1% of the viewport's larger dimension
	LenCm   CSSLength = iota // centimeters -- 1cm = 96px/2.54
	LenMm   CSSLength = iota // millimeters -- 1mm = 1/10th of cm
	LenQ    CSSLength = iota // quarter-millimeters -- 1q = 1/40th of cm
	LenIn   CSSLength = iota // inches -- 1in = 2.54cm = 96px
	LenPc   CSSLength = iota // picas -- 1pc = 1/6th of 1in
	LenPt   CSSLength = iota // points -- 1pt = 1/72th of 1in
	LenPx   CSSLength = iota // pixels -- 1px = 1/96th of 1in
)

// contrary to some docs, apparently need to run go generate manually
//go:generate stringer -type=CSSLength
