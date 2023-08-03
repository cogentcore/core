// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lms

import "github.com/goki/ki/kit"

// Opponents enumerates the three primary opponency channels:
// WhiteBlack, RedGreen, BlueYellow
// using colloquial "everyday" terms.
type Opponents int

//go:generate stringer -type=Opponents

var TypeOpponents = kit.Enums.AddEnum(OpponentsN, kit.NotBitFlag, nil)

func (ev Opponents) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Opponents) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

const (
	// White vs. Black greyscale
	WhiteBlack Opponents = iota

	// Red vs. Green
	RedGreen

	// Blue vs. Yellow
	BlueYellow

	// number of opponents
	OpponentsN
)
