// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The cases of these tests are those of the tests of the path intersections
// of https://github.com/tdewolff/canvas (path_intersection_test.go), at
// commit 0980464, with short coordinates (those with many digits need the
// precision of float64).

package intersect

import (
	"fmt"
	"testing"

	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
)

// assertPath asserts that the path is the given SVG path, within ppath.Epsilon.
func assertPath(t *testing.T, want string, got ppath.Path, msg ...any) {
	t.Helper()
	w := ppath.MustParseSVGPath(want)
	assert.True(t, w.Equals(got), "want %s, got %s %v", w.ToSVG(), got.ToSVG(), fmt.Sprint(msg...))
}

func TestCanvasSettle(t *testing.T) {
	tts := []struct {
		fillRule ppath.FillRules
		p, r     string
	}{
		{ppath.NonZero, "L2 0L2 2L0 2z", "L2 0L2 2L0 2z"}, // ccw
		{ppath.NonZero, "L0 2L2 2L2 0z", "L2 0L2 2L0 2z"}, // !ccw
		{ppath.NonZero, "L10 10L10 0L0 10z", "L5 5L0 10zM5 5L10 0L10 10z"},
		{ppath.NonZero, "L10 10L0 10L10 0z", "L10 0L5 5zM0 10L5 5L10 10z"},
		{ppath.NonZero, "L10 10L20 0L20 10L10 0L0 10z", "L5 5L0 10zM5 5L10 0L15 5L10 10zM15 5L20 0L20 10z"},
		{ppath.NonZero, "M0 2L6 2L4 4L1 1L5 1L2 4z", "M0 2L2 2L1 1L5 1L4 2L6 2L4 4L3 3L2 4z"},               // ccw
		{ppath.NonZero, "M0 2L2 4L5 1L1 1L4 4L6 2z", "M0 2L2 2L1 1L5 1L4 2L6 2L4 4L3 3L2 4z"},               // !ccw
		{ppath.EvenOdd, "M0 2L6 2L4 4L1 1L5 1L2 4z", "M0 2L2 2L3 3L2 4zM1 1L5 1L4 2L2 2zM3 3L4 2L6 2L4 4z"}, // ccw
		{ppath.EvenOdd, "M0 2L2 4L5 1L1 1L4 4L6 2z", "M0 2L2 2L3 3L2 4zM1 1L5 1L4 2L2 2zM3 3L4 2L6 2L4 4z"}, // !ccw
		{ppath.NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L15 5L15 15L5 15L5 10L0 10z"},
		{ppath.NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L8 2L8 8L2 8z", "L4 0L4 2L8 2L8 8L6 8L6 10L0 10z"},                          // ccwA  ccwB
		{ppath.NonZero, "L4 0L4 5L6 5L6 10L0 10zM2 2L2 8L8 8L8 2z", "L4 0L4 2L2 2L2 8L6 8L6 10L0 10zM4 2L8 2L8 8L6 8L6 5L4 5z"}, // ccwA !ccwB
		{ppath.NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L8 2L8 8L2 8z", "L4 0L4 2L2 2L2 8L6 8L6 10L0 10zM4 2L8 2L8 8L6 8L6 5L4 5z"}, // !ccwA  ccwB
		{ppath.NonZero, "L0 10L6 10L6 5L4 5L4 0zM2 2L2 8L8 8L8 2z", "L4 0L4 2L8 2L8 8L6 8L6 10L0 10z"},                          // !ccwA !ccwB
		{ppath.NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L8 2L8 8L2 8z", "L6 0L6 2L8 2L8 8L4 8L4 10L0 10z"},                          // ccwA  ccwB
		{ppath.NonZero, "L6 0L6 5L4 5L4 10L0 10zM2 2L2 8L8 8L8 2z", "L6 0L6 2L2 2L2 8L4 8L4 10L0 10zM4 5L6 5L6 2L8 2L8 8L4 8z"}, // ccwA !ccwB
		{ppath.NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L8 2L8 8L2 8z", "L6 0L6 2L2 2L2 8L4 8L4 10L0 10zM4 5L6 5L6 2L8 2L8 8L4 8z"}, // !ccwA  ccwB
		{ppath.NonZero, "L0 10L4 10L4 5L6 5L6 0zM2 2L2 8L8 8L8 2z", "L6 0L6 2L8 2L8 8L4 8L4 10L0 10z"},                          // !ccwA !ccwB
		{ppath.NonZero, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L15 5L15 15L5 15L5 10L0 10z"},
		{ppath.EvenOdd, "L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", "L10 0L10 5L5 5L5 10L0 10zM5 10L10 10L10 5L15 5L15 15L5 15z"},
		{ppath.NonZero, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M-1 1L0 1L0 0L4 0L4 1L5 1L5 1.5L5.5 1.5L5.5 2.5L5 2.5L5 3L4 3L4 4L0 4L0 3L-1 3z"},
		{ppath.EvenOdd, "L4 0L4 4L0 4zM-1 1L1 1L1 3L-1 3zM3 1L5 1L5 3L3 3zM4.5 1.5L5.5 1.5L5.5 2.5L4.5 2.5z", "M-1 1L0 1L0 3L-1 3zM0 0L4 0L4 1L3 1L3 3L4 3L4 4L0 4L0 3L1 3L1 1L0 1zM4 1L5 1L5 1.5L4.5 1.5L4.5 2.5L5 2.5L5 3L4 3zM5 1.5L5.5 1.5L5.5 2.5L5 2.5z"},
		{ppath.NonZero, "L10 0L10 30L0 30zM1 1L1 9L9 9L9 1zM1 11L1 19L9 19L9 11z", "M0 0L10 0L10 30L0 30zM1 1L1 9L9 9L9 1zM1 11L1 19L9 19L9 11z"},
		{ppath.NonZero, "L5 5L10 0L10 10L5 5L0 10z", "L5 5L0 10zM5 5L10 0L10 10z"},
		{ppath.NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 -1L1 -1z", "M0 0L1 0L1 -1L4 -1L4 0L2 2z"},
		{ppath.NonZero, "L2 2L3 0zM1 0L2 2L4 0L4 3L1 3z", "L1 0L1 1zM1 0L3 0L2 2zM1 1L2 2L4 0L4 3L1 3z"},
		{ppath.NonZero, "L1 0L1 1L0 1zM1 0L1 1L2 1L2 0z", "L2 0L2 1L0 1z"},
		{ppath.NonZero, "L1 0L1 1L0 1zM1 0L2 0L2 1L1 1z", "L2 0L2 1L0 1z"},
		{ppath.NonZero, "L10 0L5 2L5 8L0 10L10 10L5 8L5 2z", "L10 0L5 2zM0 10L5 8L10 10z"},
		{ppath.NonZero, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", "L10 0L10 15L0 15z"},
		{ppath.EvenOdd, "L10 0L10 10L5 7.5L10 5L10 15L0 15z", "L10 0L10 5L5 7.5L10 10L10 15L0 15z"},
		{ppath.NonZero, "L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "L10 0L10 5L5 7.5L10 10L10 15L0 15z"},
		{ppath.EvenOdd, "L10 0L10 5L0 10L0 5L10 10L10 15L0 15z", "L10 0L10 5L5 7.5L0 5zM0 10L5 7.5L10 10L10 15L0 15z"},
		{ppath.NonZero, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "L10 0L10 5L7.5 7.5L5 5L2.5 7.5L0 5zM0 10L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z"},
		{ppath.EvenOdd, "L10 0L10 5L5 10L0 5L0 15L10 15L10 10L5 5L0 10z", "L10 0L10 5L7.5 7.5L5 5L2.5 7.5L0 5zM0 10L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z"},
		{ppath.NonZero, "L3 0L3 1L0 1zM1 0L1 1L2 1L2 0z", "L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},
		{ppath.Positive, "L2 0L1 0L3 0L1.5 1z", "L3 0L1.5 1z"},
		{ppath.Positive, "L0 2L0 1L0 3L-1 1.5z", "M-1 1.5L0 0L0 3z"},
		{ppath.Positive, "L2 0L1 0L3 0z", ""},
		{ppath.Positive, "L0 2L0 1L0 3z", ""},
		{ppath.NonZero, "M1 4L0 2L1 0L2 0L2 4zM1 3L0 2L1 1z", "M0 2L1 0L2 0L2 4L1 4z"},                                                                    // tangent left-most endpoint
		{ppath.NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 1L1 3z", "M0 2L1 0L2 0L2 4L1 4z"},                                                                    // tangent left-most endpoint
		{ppath.NonZero, "M0 2L1 0L2 0L2 4L1 4zM0 2L1 3L1 1z", "M0 2L1 0L2 0L2 4L1 4L0 2L1 3L1 1z"},                                                        // tangent left-most endpoint
		{ppath.NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 1L2 3L1 4z", "M0 2L1 0L2 1L1.5 2L2 3L1 4z"},                                                              // secant left-most endpoint
		{ppath.NonZero, "M0 2L1 0L2 1L1 3zM0 2L1 4L2 3L1 1z", "M0 2L1 0L2 1L1.5 2L1 1zM0 2L1 3L1.5 2L2 3L1 4z"},                                           // secant left-most endpoint
		{ppath.NonZero, "L2 0L2 2L0 2L0 1L-1 2L-2 2L-1 2L0 1z", "L2 0L2 2L0 2z"},                                                                          // parallel left-most endpoint
		{ppath.NonZero, "L0 1L-1 2L0 1z", ""},                                                                                                             // all parallel
		{ppath.NonZero, "M1 0L1 2.1L1.1 2.1L1.1 2L0 2L0 3L2 3L2 1L5 1L5 2L5.1 2.1L5.1 1.9L5 2L5 3L6 3L6 0z", "M0 2L1 2L1 0L6 0L6 3L5 3L5 1L2 1L2 3L0 3z"}, // CCW open first leg for CW path
		{ppath.Negative, "M2 1L0 0L0 2L2 1L1 0L1 2z", "L1 0.5L1 0L2 1L1 2L1 1.5L0 2z"},
		{ppath.Positive, "M0 -1L10 -1L10 1L5 1L5 -1L10 -1L10 1L0 1z", "M0 -1L10 -1L10 1L0 1z"},
	}
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			assertPath(t, tt.r, Settle(ppath.MustParseSVGPath(tt.p), tt.fillRule))
		})
	}
}

func TestCanvasAnd(t *testing.T) {
	tts := []struct {
		p, q, r string
	}{
		{"L0 10L10 10L10 0z", "", ""},
		{"", "L0 10L10 10L10 0z", ""},
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L5 10z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "M2.5 5L7.5 5L5 10z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "M2.5 5L7.5 5L5 10z"},
		{"L10 0L5 10z", "M0 10L10 10L5 20z", ""},
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", ""},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", ""},
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "M2 2L8 2L5 8z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M2 2L8 2L5 8z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "L1 0L1 1L0 1z"},
		{"L2 0L2 1L0 1z", "L0 1L1 1L1 0z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "L1 0L1 1L0 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L1 1L2 1L2 0z", "M1 0L2 0L2 1L1 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 2L0 2z", "L1 0L1 1L0 1z"},
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 0L5 10z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L5 10L10 0z", "L5 10L10 0z", "L10 0L5 10z"},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", "L10 -10L20 0L10 10z"},
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 4L6 6L6 7L4 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M4 3L6 3L6 4L4 6z"},
		{"M1 3L6 3L6 4L4 6L5 7L1 7z", "M9 3L4 3L4 6L5 7L9 7z", "M4 3L6 3L6 4L4 6z"},
		{"L3 0L3 1L0 1z", "M1 -0.1L2 -0.1L2 1.1L1 1.1z", "M1 0L2 0L2 1L1 1z"},
		{"L10 0L10 10L0 10z", "M2 0L8 0L8 6L 2 6z", "M2 0L8 0L8 6L 2 6z"},
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "M1 2L2 2L2 3L1 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", ""},
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", ""},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 -0.1L0.4 0.5L1 1.1L1 2L0 2z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 0L0.5 0.5L1 1L1 2L0 2z"},
		{"M1 0L2 0L2 1L1 1zM0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z", "M0 -1L1 -1L1 2L0 2z", "M0 -1L1 -1L1 0.1L0.6 0.5L1 0.9L1 2L0 2z"},
		{"L1 0L1.1 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0zM1 0L1.1 0.5L1 1z"},
		{"L1 0L0.9 0.5L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "M1 -1L2 -1L2 2L1 2L1 1L1.5 0.5L1 0z"},
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 1L3 1L3 2L2 2L2 3L1 3zM2 3L3 3L3 4L2 4z"},                                      // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "M1 0L2 0L2 1L1 1zM1 2L2 2L2 3L1 3z"},                                 // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "M0 2L2 2L2 3L0 3zM1 0L2 0L2 1L1 1z"},                                 // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "M0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9zM1 0L2 0L2 1L1 1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "M1 0L2 0L2 1L1 1z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z"},                                                  // two inside the same
		{"M5 1L5 9", "L10 0L10 10L0 10z", "M5 1L5 9"},                                                                                                  // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", ""},                                                                                                        // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "M5 5L5 10"},                                                                                                // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 10"},                                                                                                      // touch
		{"M5 0L10 0L10 5", "L10 0L10 10L0 10z", ""},                                                                                                    // boundary
		{"L5 0L5 5", "L10 0L10 10L0 10z", "M5 0L5 5"},                                                                                                  // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "M1 1L2 0M8 0L9 1"},                                                                                  // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", ""},                                                                                                // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                                                                                                             // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "M5 0L5 1L6 0"},                                                                                         // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},                                                                                             // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M6 0L7 1"},                                                                                             // touch with parallel
		{"M5 5L25 5", "L10 0L10 10L0 10zM20 0L30 0L30 10L20 10z", "M5 5L10 5M20 5L25 5"},                                                               // cross twice
		{"L0 -20L20 -20L20 20L0 20z", "L10 10L10 -10L-10 10L-10 -10z", "L10 -10L10 10z"},
		{"L20 0L20 20L-20 20L-20 0z", "L10 10L10 -10L-10 10L-10 -10z", "M-10 0L0 0L-10 10zM0 0L10 0L10 10z"},
		{"M-5 -5L0 -5L0 5L-5 5z", "L10 0L10 10L-10 10L-10 0L10 0L10 -10L-10 -10L-10 0z", "M-5 -5L0 -5L0 5L-5 5z"},
		{"M-5 -5L0 -5L0 5L-5 5z", "L-10 0L-10 10L10 10L10 0L-10 0L-10 -10L10 -10L10 0z", "M-5 -5L0 -5L0 5L-5 5z"},
		{"L20 0L20 20L0 20z", "L10 0L10 10L-10 10L-10 0L10 0L10 -10L-10 -10L-10 0z", "L10 0L10 10L0 10z"},
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M1 4L5 4L5 6L1 6z"},
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "M1 0L2 0L2 3L1 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "M0 1L3 1L3 2L0 2z"},
		{"M23 15L24 15L24 16L23 16zM23.4 14L24.4 14L24.4 15L23.4 15z", "M15 16A1 1 0 0 1 16 15L24 15A1 1 0 0 1 25 16L25 24A1 1 0 0 1 24 25L16 25A1 1 0 0 1 15 24z", "M23 15L24 15L24 16L23 16z"},
		{"M23 15L24 15L24 16L23 16zM24 15.4L25 15.4L25 16.4L24 16.4z", "M14 14L24 14L24 24L14 24z", "M23 15L24 15L24 16L23 16z"},
		{"M0 1L2 1L2 2L0 2zM3 1L5 1L5 2L3 2z", "M1 0L4 0L4 3L1 3z", "M1 1L2 1L2 2L1 2zM3 1L4 1L4 2L3 2z"},
		{"L5 0L5 5L0 5zM1 1L1 4L4 4L4 1z", "M3 2L6 2L6 3L3 3z", "M4 2L5 2L5 3L4 3z"},
	}
	for _, tt := range tts {
		t.Run(tt.p+"x"+tt.q, func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			q := ppath.MustParseSVGPath(tt.q)
			assertPath(t, tt.r, And(p, q))
			if p.Closed() {
				assertPath(t, tt.r, And(q, p), "swapped arguments")
			}
		})
	}
}

func TestCanvasOr(t *testing.T) {
	tts := []struct {
		p, q, r string
	}{
		{"L0 10L10 10L10 0z", "", "L10 0L10 10L0 10z"},
		{"", "L0 10L10 10L10 0z", "L10 0L10 10L0 10z"},
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L10 5L5 15L0 5L2.5 5z"},
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L4 0L4 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 1L4 1L4 3L2 3L2 2L0 2z"},
		{"L2 0L2 2L0 2zM2 0L4 0L4 2L2 2z", "M2.5 0.5L3.5 0.5L3.5 1.5L2.5 1.5z", "L4 0L4 2L0 2z"},
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 0L5 10z", "L10 0L5 10z", "L10 0L5 10z"},
		{"L10 -10L20 0L10 10z", "Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z"},
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "L2 0L2 1L0 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "L2 0L2 1L0 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L3 0L3 1L0 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "L2 0L2 2L0 2z"},
		{"M2 0L2 2L0 2L0 0z", "M2 0L4 0L4 2L0 2L0 0z", "L4 0L4 2L0 2z"},
		{"M2 2L0 2L0 0L2 0z", "M2 0L4 0L4 2L0 2L0 0z", "L4 0L4 2L0 2z"},
		{"M2 0L2 1L0 1L0 0z", "M1 0L3 0L3 1L1 1z", "L3 0L3 1L0 1z"},
		{"M2 1L0 1L0 0L2 0z", "M1 0L3 0L3 1L1 1z", "L0 0L3 0L3 1L0 1z"},
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L2 3L2 4L1 4L1 3L0 3z"},
		{"L10 0L10 5L7.5 7.5L5 5L2.5 7.5L5 10L7.5 7.5L10 10L10 15L0 15z", "M7.5 7.5L5 10L2.5 7.5L5 5z", "M0 0L10 0L10 5L7.5 7.5L10 10L10 15L0 15z"},
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M0 1L1 1L1 0L3 0L3 1L4 1L4 2L3 2L3 3L4 3L4 2L5 2L5 5L2 5L2 4L1 4L1 3L0 3z"}, // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L3 0L3 1L0 1zM0 2L3 2L3 3L0 3z"},                               // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L3 0L3 1L0 1zM0 2L2 2L2 3L0 3z"},                               // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L3 0L3 1L0 1zM0 2L2 2L2 3L0 3z"},               // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L3 0L3 1L0 1zM0 2L4 2L4 3L0 3z"},                               // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z"},                                                                 // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4z"},                                                                 // two inside the same
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 1L5 9"},                                                                           // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M0 0L10 0L10 10L0 10zM15 1L15 9"},                                                                   // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 5L5 10L5 15"},                                                                    // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM0 0L10 10"},                                                                           // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 5"},                                                                           // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 1L2 0M8 0L9 1"},                                                           // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"},                                                       // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "M0 0L10 0L10 10L0 10z"},                                                                                  // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 1L7 -1"},                                                                 // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 -1L7 1"},                                                                 // touch with parallel
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 3L5 3L5 4L6 4L5 5L6 6L5 6L5 7L1 7L1 6L0 6zM6 4L7 5L6 6z"},
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L2 0L2 3L0 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 2L0 2z"},
	}
	for _, tt := range tts {
		t.Run(tt.p+"x"+tt.q, func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			q := ppath.MustParseSVGPath(tt.q)
			assertPath(t, tt.r, Or(p, q))
		})
	}
}

func TestCanvasXor(t *testing.T) {
	tts := []struct {
		p, q, r string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5zM0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L4 0L4 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 1L4 1L4 3L2 3L2 2L0 2z"},
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10zM0 10L10 10L5 20z"},
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", "M0 0L10 0L5 10zM2 2L5 8L8 2z"},
		{"L10 0L5 10z", "L10 0L5 10z", ""},
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L9 3L9 7L6 7L6 6L4 4L4 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 6L6 4L6 3L9 3L9 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L1 0L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M0 1L1 1L1 0L2 0L2 2L0 2z"},
		{"L2 0L0 2z", "L2 2L0 2z", "L2 0L1 1zM0 2L1 1L2 2z"},
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L2 3L2 2L1 2L1 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L2 3L2 4L1 4L1 3L0 3z"},
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L1 0L1 1L0 1zM0 2L1 2L1 3L0 3zM2 0L3 0L3 1L2 1zM2 2L3 2L3 3L2 3z"},                                 // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},                                                                   // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1zM2 0L3 0L3 1L2 1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L1 0L1 1L0 1zM0 2L4 2L4 3L0 3zM2 0L3 0L3 1L2 1z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                                   // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                                   // two inside the same
		{"M5 1L5 9", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                                                                                                       // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M0 0L10 0L10 10L0 10zM15 1L15 9"},                                                                                                       // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 10L5 15"},                                                                                                            // cross
		{"L10 10", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                                                                                                         // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                                                                                                       // touch with parallel
		{"M1 1L2 0L8 0L9 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                                                                                               // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM1 -1L2 0M8 0L9 -1"},                                                                                           // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", "L10 0L10 10L0 10z"},                                                                                                                          // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM6 0L7 -1"},                                                                                                         // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "L10 0L10 10L0 10zM5 0L5 -1L6 0"},                                                                                                     // touch with parallel
		{"L2 0L2 2L4 2L4 4L2 2L0 2z", "L2 0L2 2L4 4L2 4L2 2L0 2z", "M2 2L4 2L4 4L2 4z"},
		{"L2 0L2 1.9L4 1.9L4 4L2 2L0 2z", "L2 0L2 2L4 4L1.9 4L1.9 2L0 2z", "M1.9 2L2 2L2 1.9L4 1.9L4 4L1.9 4z"}, // simple version of above
		{"L2 0L2 1L3 1L2 0L4 0L4 2L0 2z", "L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2L2 1L3 1z"},
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 6L0 6zM1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7zM5 4L6 4L5 5zM5 5L6 6L5 6zM6 4L7 5L6 6z"},
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3z"},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1z"},
	}
	for _, tt := range tts {
		t.Run(tt.p+"x"+tt.q, func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			q := ppath.MustParseSVGPath(tt.q)
			assertPath(t, tt.r, Xor(p, q))
		})
	}
}

func TestCanvasNot(t *testing.T) {
	tts := []struct {
		p, q, r string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5z"},
		{"L10 0L5 10z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5z"},
		{"L5 10L10 0z", "M0 5L5 15L10 5z", "L10 0L7.5 5L2.5 5z"},
		{"M0 5L10 5L5 15z", "L10 0L5 10z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L10 5L5 15z", "L5 10L10 0z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L5 15L10 5z", "L10 0L5 10z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"M0 5L5 15L10 5z", "L5 10L10 0z", "M0 5L2.5 5L5 10L7.5 5L10 5L5 15z"},
		{"L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z"},
		{"L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z"},
		{"M2 0L4 0L4 2L2 2z", "L2 0L2 2L0 2z", "M2 0L4 0L4 2L2 2z"},
		{"M2 1L4 1L4 3L2 3z", "L2 0L2 2L0 2z", "M2 1L4 1L4 3L2 3z"},
		{"L10 0L5 10z", "M0 10L10 10L5 20z", "L10 0L5 10z"},
		{"M0 10L10 10L5 20z", "L10 0L5 10z", "M0 10L10 10L5 20z"},
		{"L10 0L5 10z", "M2 2L8 2L5 8z", "L10 0L5 10zM2 2L5 8L8 2z"},
		{"M2 2L8 2L5 8z", "L10 0L5 10z", ""},
		{"L10 0L5 10z", "L10 0L5 10z", ""},
		{"L10 -10L20 0L10 10z", "A10 10 0 0 0 20 0A10 10 0 0 0 0 0z", ""},
		{"Q10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", "L10 -10L20 0L10 10z", ""},
		{"M1 3L4 3L4 4L6 6L6 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 7L1 7z"},
		{"M1 3L6 3L6 4L4 6L4 7L1 7z", "M9 3L4 3L4 7L9 7z", "M1 3L4 3L4 7L1 7z"},
		{"L2 0L2 1L0 1z", "L1 0L1 1L0 1z", "M1 0L2 0L2 1L1 1z"},
		{"L1 0L1 1L0 1z", "L2 0L2 1L0 1z", ""},
		{"L3 0L3 1L0 1z", "M1 0L2 0L2 1L1 1z", "L1 0L1 0L1 1L0 1zM2 0L3 0L3 1L2 1z"},
		{"L2 0L2 2L0 2z", "L1 0L1 1L0 1z", "M0 1L1 1L1 0L2 0L2 2L0 2z"},
		{"L3 0L3 3L0 3z", "M1 2L2 2L2 3L1 3z", "L3 0L3 3L2 3L2 2L1 2L1 3L0 3z"},
		{"L3 0L3 3L0 3z", "M1 3L2 3L2 4L1 4z", "L3 0L3 3L0 3z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1.1L1.6 0.5L1 -0.1L1 -1z", "M1 -0.1L1.6 0.5L1 1.1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 1L1.5 0.5L1 0L1 -1z", "M1 0L1.5 0.5L1 1z"},
		{"M2 -1L2 2L1 2L1 -1z", "L1 0L1 1L0 1zM2 -1L2 2L1 2L1 0.9L1.4 0.5L1 0.1L1 -1z", "M1 0.1L1.4 0.5L1 0.9z"},
		{"M1 0L3 0L3 4L1 4z", "M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 1L1 1zM1 3L2 3L2 4L1 4zM2 2L3 2L3 3L2 3z"},                                          // different winding
		{"M0 1L4 1L4 3L0 3zM2 2L2 5L5 5L5 2z", "M1 0L3 0L3 4L1 4z", "M0 1L1 1L1 3L0 3zM2 4L3 4L3 3L4 3L4 2L5 2L5 5L2 5zM3 1L4 1L4 2L3 2z"},                          // different winding
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM1 2L3 2L3 3L1 3z", "L1 0L1 1L0 1zM0 2L1 2L1 3L0 3z"},                                                  // two overlapping
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0 2L2 2L2 3L0 3z", "L1 0L1 1L0 1z"},                                                                   // one overlapping, one equal
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM0.1 2.1L1.9 2.1L1.9 2.9L0.1 2.9z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3zM0.1 2.1L0.1 2.9L1.9 2.9L1.9 2.1z"}, // one overlapping, one inside the other
		{"L2 0L2 1L0 1zM0 2L2 2L2 3L0 3z", "M1 0L3 0L3 1L1 1zM2 2L4 2L4 3L2 3z", "L1 0L1 1L0 1zM0 2L2 2L2 3L0 3z"},                                                  // one overlapping, the others separate
		{"L7 0L7 4L0 4z", "M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4zM1 1L1 3L3 3L3 1zM4 1L4 3L6 3L6 1z"},                                                  // two inside the same
		{"M1 1L3 1L3 3L1 3zM4 1L6 1L6 3L4 3z", "L7 0L7 4L0 4z", ""},                                                                                                 // two inside the same
		{"M5 1L5 9", "L10 0L10 10L0 10z", ""},                                                                                                                       // in
		{"M15 1L15 9", "L10 0L10 10L0 10z", "M15 1L15 9"},                                                                                                           // out
		{"M5 5L5 15", "L10 0L10 10L0 10z", "M5 10L5 15"},                                                                                                            // cross
		{"L10 10", "L10 0L10 10L0 10z", ""},                                                                                                                         // touch
		{"L5 0L5 5", "L10 0L10 10L0 10z", ""},                                                                                                                       // touch with parallel
		{"M1 1L2 0L8 0L9 9", "L10 0L10 10L0 10z", ""},                                                                                                               // touch with parallel
		{"M1 -1L2 0L8 0L9 -1", "L10 0L10 10L0 10z", "M1 -1L2 0M8 0L9 -1"},                                                                                           // touch with parallel
		{"L10 0", "L10 0L10 10L0 10z", ""},                                                                                                                          // touch with parallel
		{"L5 0L5 1L7 -1", "L10 0L10 10L0 10z", "M6 0L7 -1"},                                                                                                         // touch with parallel
		{"L5 0L5 -1L7 1", "L10 0L10 10L0 10z", "M5 0L5 -1L6 0"},                                                                                                     // touch with parallel
		{"M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 7L1 7z", "M0 4L1 4L1 6L0 6zM5 4L6 4L5 5zM5 5L6 6L5 6zM6 4L7 5L6 6z"},
		{"M1 3L5 3L5 7L1 7z", "M0 4L6 4L6 6L0 6zM5 5L6 6L7 5L6 4z", "M1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7z"},
		{"M1 3L5 3L5 7L1 7zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7z", "M0 4L6 4L6 6L0 6zM5 5A1 1 0 0 0 7 5A1 1 0 0 0 5 5zM7.2 7A0.8 0.8 0 0 1 8.8 7A0.8 0.8 0 0 1 7.2 7zM7.2 3A0.8 0.8 0 0 1 8.8 3A0.8 0.8 0 0 1 7.2 3z", "M1 3L5 3L5 4L1 4zM1 6L5 6L5 7L1 7z"},
		{"L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", "M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3z"},
		{"M1 0L2 0L2 3L1 3z", "L1 0L1 3L0 3zM1 0L2 0L2 2L1 2zM1 1L2 1L2 3L1 3z", ""},
		{"L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", "M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1z"},
		{"M0 1L3 1L3 2L0 2z", "L3 0L3 1L0 1zM0 1L2 1L2 2L0 2zM1 1L3 1L3 2L1 2z", ""},
	}
	for _, tt := range tts {
		t.Run(tt.p+"x"+tt.q, func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			q := ppath.MustParseSVGPath(tt.q)
			assertPath(t, tt.r, Not(p, q))
		})
	}
}

func TestCanvasDivideBy(t *testing.T) {
	tts := []struct {
		p, q, r string
	}{
		{"L10 0L5 10z", "M0 5L10 5L5 15z", "L10 0L7.5 5L2.5 5zM2.5 5L7.5 5L5 10z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L5 1L5 3L1 3z", "M0 0L2 0L2 1L1 1L1 2L0 2zM1 1L2 1L2 2L1 2zM4 0L6 0L6 2L5 2L5 1L4 1zM4 1L5 1L5 2L4 2z"},
		{"L2 0L2 2L0 2zM4 0L6 0L6 2L4 2z", "M1 1L1 3L5 3L5 1z", "M0 0L2 0L2 1L1 1L1 2L0 2zM1 1L2 1L2 2L1 2zM4 0L6 0L6 2L5 2L5 1L4 1zM4 1L5 1L5 2L4 2z"},
	}
	for _, tt := range tts {
		t.Run(tt.p+"x"+tt.q, func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			q := ppath.MustParseSVGPath(tt.q)
			assertPath(t, tt.r, DivideBy(p, q))
		})
	}
}
