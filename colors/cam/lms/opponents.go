// Copyright (c) 2021, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lms

//go:generate core generate

// OpponentValues holds color opponent values based on cone-like L,M,S inputs
// These values are useful for generating inputs to vision models that
// simulate color opponency representations in the brain.
type OpponentValues struct {

	// red vs. green (long vs. medium)
	RedGreen float32

	// blue vs. yellow (short vs. avg(long, medium))
	BlueYellow float32

	// greyscale luminance channel -- typically use L* from LAB as best
	Grey float32
}

// NewOpponentValues returns a new [OpponentValues] from values representing
// the LMS long, medium, short cone responses, and an overall grey value.
func NewOpponentValues(l, m, s, lm, grey float32) OpponentValues {
	return OpponentValues{RedGreen: l - m, BlueYellow: s - lm, Grey: grey}
}

// Opponents enumerates the three primary opponency channels:
// [WhiteBlack], [RedGreen], and [BlueYellow] using colloquial
// "everyday" terms.
type Opponents int32 //enums:enum

const (
	// White vs. Black greyscale
	WhiteBlack Opponents = iota

	// Red vs. Green
	RedGreen

	// Blue vs. Yellow
	BlueYellow
)
