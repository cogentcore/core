// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textpos

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWordAt(t *testing.T) {
	txt := []rune(`assert.Equal(t, 1, s.Decoration.NumColors())`)

	tests := []struct {
		pos int
		rg  Range
	}{
		{0, Range{0, 6}},
		{5, Range{0, 6}},
		{6, Range{6, 12}},
		{8, Range{7, 12}},
		{12, Range{12, 14}},
		{13, Range{13, 14}},
		{14, Range{14, 17}},
		{15, Range{15, 17}},
		{16, Range{16, 17}},
		{20, Range{20, 31}},
		{21, Range{21, 31}},
		{43, Range{43, 44}},
		{42, Range{42, 44}},
		{41, Range{41, 44}},
		{40, Range{32, 41}},
		{50, Range{43, 44}},
	}
	for _, test := range tests {
		rg := WordAt(txt, test.pos)
		// fmt.Println(test.pos, rg, string(txt[rg.Start:rg.End]))
		assert.Equal(t, test.rg, rg)
	}
}

func TestForwardWord(t *testing.T) {
	txt := []rune(`assert.Equal(t, 1, s.Decoration.NumColors())`)

	tests := []struct {
		pos   int
		steps int
		wpos  int
		nstep int
	}{
		{0, 1, 6, 1},
		{1, 1, 6, 1},
		{6, 1, 12, 1},
		{7, 1, 12, 1},
		{0, 2, 12, 2},
		{40, 1, 41, 1},
		{40, 2, 43, 2},
		{42, 1, 43, 1},
		{43, 1, 43, 0},
	}
	for _, test := range tests {
		wp, ns := ForwardWord(txt, test.pos, test.steps)
		// fmt.Println(test.pos, test.steps, wp, ns, string(txt[test.pos:wp]))
		assert.Equal(t, test.wpos, wp)
		assert.Equal(t, test.nstep, ns)
	}
}

func TestBackwardWord(t *testing.T) {
	txt := []rune(`assert.Equal(t, 1, s.Decoration.NumColors())`)

	tests := []struct {
		pos   int
		steps int
		wpos  int
		nstep int
	}{
		{0, 1, 0, 0},
		{1, 1, 0, 1},
		{5, 1, 0, 1},
		{6, 1, 0, 1},
		{6, 2, 0, 1},
		{7, 1, 6, 1},
		{8, 1, 6, 1},
		{9, 1, 6, 1},
		{9, 2, 0, 2},
	}
	for _, test := range tests {
		wp, ns := BackwardWord(txt, test.pos, test.steps)
		// fmt.Println(test.pos, test.steps, wp, ns, string(txt[wp:test.pos]))
		assert.Equal(t, test.wpos, wp)
		assert.Equal(t, test.nstep, ns)
	}
}
