// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"fmt"
	"strings"
	"testing"

	"cogentcore.org/core/base/tolassert"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/stretchr/testify/assert"
)

func tolEqualVec2(t *testing.T, a, b math32.Vector2, tols ...float64) {
	tol := 1.0e-4
	if len(tols) == 1 {
		tol = tols[0]
	}
	assert.InDelta(t, b.X, a.X, tol)
	assert.InDelta(t, b.Y, a.Y, tol)
}

func tolEqualBox2(t *testing.T, a, b math32.Box2, tols ...float64) {
	tol := 1.0e-4
	if len(tols) == 1 {
		tol = tols[0]
	}
	tolEqualVec2(t, b.Min, a.Min, tol)
	tolEqualVec2(t, b.Max, a.Max, tol)
}

func TestPathCrossingsWindings(t *testing.T) {
	var tts = []struct {
		p         string
		pos       math32.Vector2
		crossings int
		windings  int
		boundary  bool
	}{
		// within bbox of segment
		{"L10 10", math32.Vector2{2.0, 5.0}, 1, 1, false},
		{"L-10 10", math32.Vector2{-2.0, 5.0}, 0, 0, false},
		{"Q10 5 0 10", math32.Vector2{2.0, 5.0}, 1, 1, false},
		{"Q-10 5 0 10", math32.Vector2{-2.0, 5.0}, 0, 0, false},
		{"C10 0 10 10 0 10", math32.Vector2{2.0, 5.0}, 1, 1, false},
		{"C-10 0 -10 10 0 10", math32.Vector2{-2.0, 5.0}, 0, 0, false},
		{"A5 5 0 0 1 0 10", math32.Vector2{2.0, 5.0}, 1, 1, false},
		{"A5 5 0 0 0 0 10", math32.Vector2{-2.0, 5.0}, 0, 0, false},
		{"L10 0L10 10L0 10z", math32.Vector2{5.0, 5.0}, 1, 1, false},  // mid
		{"L0 10L10 10L10 0z", math32.Vector2{5.0, 5.0}, 1, -1, false}, // mid

		// on boundary
		{"L10 0L10 10L0 10z", math32.Vector2{0.0, 5.0}, 1, 0, true},   // left
		{"L10 0L10 10L0 10z", math32.Vector2{10.0, 5.0}, 0, 0, true},  // right
		{"L10 0L10 10L0 10z", math32.Vector2{0.0, 0.0}, 0, 0, true},   // bottom-left
		{"L10 0L10 10L0 10z", math32.Vector2{5.0, 0.0}, 0, 0, true},   // bottom
		{"L10 0L10 10L0 10z", math32.Vector2{10.0, 0.0}, 0, 0, true},  // bottom-right
		{"L10 0L10 10L0 10z", math32.Vector2{0.0, 10.0}, 0, 0, true},  // top-left
		{"L10 0L10 10L0 10z", math32.Vector2{5.0, 10.0}, 0, 0, true},  // top
		{"L10 0L10 10L0 10z", math32.Vector2{10.0, 10.0}, 0, 0, true}, // top-right
		{"L0 10L10 10L10 0z", math32.Vector2{0.0, 5.0}, 1, 0, true},   // left
		{"L0 10L10 10L10 0z", math32.Vector2{10.0, 5.0}, 0, 0, true},  // right
		{"L0 10L10 10L10 0z", math32.Vector2{0.0, 0.0}, 0, 0, true},   // bottom-left
		{"L0 10L10 10L10 0z", math32.Vector2{5.0, 0.0}, 0, 0, true},   // bottom
		{"L0 10L10 10L10 0z", math32.Vector2{10.0, 0.0}, 0, 0, true},  // bottom-right
		{"L0 10L10 10L10 0z", math32.Vector2{0.0, 10.0}, 0, 0, true},  // top-left
		{"L0 10L10 10L10 0z", math32.Vector2{5.0, 10.0}, 0, 0, true},  // top
		{"L0 10L10 10L10 0z", math32.Vector2{10.0, 10.0}, 0, 0, true}, // top-right

		// outside
		{"L10 0L10 10L0 10z", math32.Vector2{-1.0, 0.0}, 0, 0, false},    // bottom-left
		{"L10 0L10 10L0 10z", math32.Vector2{-1.0, 5.0}, 2, 0, false},    // left
		{"L10 0L10 10L0 10z", math32.Vector2{-1.0, 10.0}, 0, 0, false},   // top-left
		{"L10 0L10 10L0 10z", math32.Vector2{11.0, 0.0}, 0, 0, false},    // bottom-right
		{"L10 0L10 10L0 10z", math32.Vector2{11.0, 5.0}, 0, 0, false},    // right
		{"L10 0L10 10L0 10z", math32.Vector2{11.0, 10.0}, 0, 0, false},   // top-right
		{"L0 10L10 10L10 0z", math32.Vector2{-1.0, 0.0}, 0, 0, false},    // bottom-left
		{"L0 10L10 10L10 0z", math32.Vector2{-1.0, 5.0}, 2, 0, false},    // left
		{"L0 10L10 10L10 0z", math32.Vector2{-1.0, 10.0}, 0, 0, false},   // top-left
		{"L0 10L10 10L10 0z", math32.Vector2{11.0, 0.0}, 0, 0, false},    // bottom-right
		{"L0 10L10 10L10 0z", math32.Vector2{11.0, 5.0}, 0, 0, false},    // right
		{"L0 10L10 10L10 0z", math32.Vector2{11.0, 10.0}, 0, 0, false},   // top-right
		{"L10 0L10 10L0 10L1 5z", math32.Vector2{0.0, 5.0}, 2, 0, false}, // left over endpoints

		// subpath
		{"L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z", math32.Vector2{1.0, 1.0}, 1, 1, false},
		{"L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z", math32.Vector2{3.0, 3.0}, 2, 2, false},
		{"L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z", math32.Vector2{3.0, 3.0}, 2, 0, false},
		{"L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z", math32.Vector2{0.0, 0.0}, 0, 0, true},
		{"L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z", math32.Vector2{2.0, 2.0}, 1, 1, true},
		{"L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z", math32.Vector2{2.0, 2.0}, 1, 1, true},
		{"L10 0L10 10L0 10zM5 5L15 5L15 15L5 15z", math32.Vector2{7.5, 5.0}, 1, 1, true},

		// on segment end
		{"L5 -5L10 0L5 5z", math32.Vector2{5.0, 0.0}, 1, 1, false},                      // mid
		{"L5 -5L10 0L5 5z", math32.Vector2{0.0, 0.0}, 1, 0, true},                       // left
		{"L5 -5L10 0L5 5z", math32.Vector2{10.0, 0.0}, 0, 0, true},                      // right
		{"L5 -5L10 0L5 5z", math32.Vector2{5.0, 5.0}, 0, 0, true},                       // top
		{"L5 -5L10 0L5 5z", math32.Vector2{5.0, -5.0}, 0, 0, true},                      // bottom
		{"L5 5L10 0L5 -5z", math32.Vector2{5.0, 0.0}, 1, -1, false},                     // mid
		{"L5 5L10 0L5 -5z", math32.Vector2{0.0, 0.0}, 1, 0, true},                       // left
		{"L5 5L10 0L5 -5z", math32.Vector2{10.0, 0.0}, 0, 0, true},                      // right
		{"L5 5L10 0L5 -5z", math32.Vector2{5.0, 5.0}, 0, 0, true},                       // top
		{"L5 5L10 0L5 -5z", math32.Vector2{5.0, -5.0}, 0, 0, true},                      // bottom
		{"M10 0A5 5 0 0 0 0 0A5 5 0 0 0 10 0z", math32.Vector2{5.0, 0.0}, 1, -1, false}, // mid
		{"M10 0A5 5 0 0 0 0 0A5 5 0 0 0 10 0z", math32.Vector2{0.0, 0.0}, 1, 0, true},   // left
		{"M10 0A5 5 0 0 0 0 0A5 5 0 0 0 10 0z", math32.Vector2{10.0, 0.0}, 0, 0, true},  // right
		{"M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z", math32.Vector2{5.0, 0.0}, 1, 0, false},  // mid
		{"M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z", math32.Vector2{0.0, 0.0}, 1, 0, true},   // left
		{"M10 0A5 5 0 0 1 0 0A5 5 0 0 1 10 0z", math32.Vector2{10.0, 0.0}, 0, 0, true},  // right

		// cross twice
		{"L10 10L10 -10L-10 10L-10 -10z", math32.Vector2{0.0, 0.0}, 1, 0, true},
		{"L10 10L10 -10L-10 10L-10 -10z", math32.Vector2{-1.0, 0.0}, 3, 1, false},
		{"L10 10L10 -10L-10 10L20 40L20 -40L-10 -10z", math32.Vector2{0.0, 0.0}, 2, 0, true},
		{"L10 10L10 -10L-10 10L20 40L20 -40L-10 -10z", math32.Vector2{1.0, 0.0}, 2, -2, false},
		{"L10 10L10 -10L-10 10L20 40L20 -40L-10 -10z", math32.Vector2{-1.0, 0.0}, 4, 0, false},

		// bugs
		{"M0 35.43000000000029L0 385.82000000000016L11.819999999999709 397.6300000000001L185.03999999999996 397.6300000000001L188.97999999999956 393.7000000000003L196.85000000000036 385.8299999999999L196.85000000000036 19.68000000000029", math32.Vector2{89.4930000000003, 19.68000000000019}, 0, 0, false}, // #346
		{"M0 35.43000000000029L0 385.82000000000016L11.819999999999709 397.6300000000001L185.03999999999996 397.6300000000001L188.97999999999956 393.7000000000003L196.85000000000036 385.8299999999999L196.85000000000036 19.68000000000029", math32.Vector2{89.4930000000003, 20}, 1, -1, false},               // #346
	}
	for _, tt := range tts {
		t.Run(fmt.Sprint(tt.p, " at ", tt.pos), func(t *testing.T) {
			p := ppath.MustParseSVGPath(tt.p)
			crossings, boundary1 := Crossings(p, tt.pos.X, tt.pos.Y)
			windings, boundary2 := Windings(p, tt.pos.X, tt.pos.Y)
			assert.Equal(t, []any{tt.crossings, tt.windings, tt.boundary, tt.boundary}, []any{crossings, windings, boundary1, boundary2})
		})
	}
}

func TestPathCCW(t *testing.T) {
	var tts = []struct {
		p   string
		ccw bool
	}{
		{"L10 0L10 10z", true},
		{"L10 0L10 -10z", false},
		{"L10 0", true},
		{"M10 0", true},
		{"Q0 -1 1 0", true},
		{"Q0 1 1 0", false},
		{"C0 -1 1 -1 1 0", true},
		{"C0 1 1 1 1 0", false},
		{"A1 1 0 0 1 2 0", true},
		{"A1 1 0 0 0 2 0", false},

		// parallel on right-most endpoint
		//{"L10 0L5 0L2.5 5z", true}, // TODO: overlapping segments?
		{"L0 5L5 5A5 5 0 0 1 0 0z", false},
		{"Q0 5 5 5A5 5 0 0 1 0 0z", false},
		{"M0 10L0 5L5 5A5 5 0 0 0 0 10z", true},
		{"M0 10Q0 5 5 5A5 5 0 0 0 0 10z", true},

		// bugs
		{"M0.31191406250000003 0.9650390625L0.3083984375 0.9724609375L0.3013671875 0.9724609375L0.29824218750000003 0.9646484375z", true},
	}
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			assert.Equal(t, tt.ccw, CCW(ppath.MustParseSVGPath(tt.p)))
		})
	}
}

func TestPathFilling(t *testing.T) {
	var tts = []struct {
		p       string
		filling []bool
		rule    ppath.FillRules
	}{
		{"M0 0", []bool{}, ppath.NonZero},
		{"L10 10z", []bool{true}, ppath.NonZero},
		{"C5 0 10 5 10 10z", []bool{true}, ppath.NonZero},
		{"C0 5 5 10 10 10z", []bool{true}, ppath.NonZero},
		{"Q10 0 10 10z", []bool{true}, ppath.NonZero},
		{"Q0 10 10 10z", []bool{true}, ppath.NonZero},
		{"A10 10 0 0 1 10 10z", []bool{true}, ppath.NonZero},
		{"A10 10 0 0 0 10 10z", []bool{true}, ppath.NonZero},

		// subpaths
		{"L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z", []bool{true, true}, ppath.NonZero},  // outer CCW,inner CCW
		{"L10 0L10 10L0 10zM2 2L8 2L8 8L2 8z", []bool{true, false}, ppath.EvenOdd}, // outer CCW,inner CCW
		{"L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z", []bool{true, false}, ppath.NonZero}, // outer CCW,inner CW
		{"L10 0L10 10L0 10zM2 2L2 8L8 8L8 2z", []bool{true, false}, ppath.EvenOdd}, // outer CCW,inner CW
		{"L10 10L0 20zM2 4L8 10L2 16z", []bool{true, true}, ppath.NonZero},         // outer CCW,inner CW
		{"L10 10L0 20zM2 4L8 10L2 16z", []bool{true, false}, ppath.EvenOdd},        // outer CCW,inner CW
		{"L10 10L0 20zM2 4L2 16L8 10z", []bool{true, false}, ppath.NonZero},        // outer CCW,inner CCW
		{"L10 10L0 20zM2 4L2 16L8 10z", []bool{true, false}, ppath.EvenOdd},        // outer CCW,inner CCW

		// paths touch at ray
		{"L10 10L0 20zM2 4L10 10L2 16z", []bool{true, true}, ppath.NonZero},  // inside
		{"L10 10L0 20zM2 4L10 10L2 16z", []bool{true, false}, ppath.EvenOdd}, // inside
		{"L10 10L0 20zM2 4L2 16L10 10z", []bool{true, false}, ppath.NonZero}, // inside
		{"L10 10L0 20zM2 4L2 16L10 10z", []bool{true, false}, ppath.EvenOdd}, // inside
		//{"L10 10L0 20zM2 2L2 18L10 10z", []bool{true, false}, NonZero},    // inside // TODO
		{"L10 10L0 20zM-1 -2L-1 22L10 10z", []bool{false, true}, ppath.NonZero}, // encapsulates
		//{"L10 10L0 20zM-2 -2L-2 22L10 10z", []bool{false, true}, NonZero}, // encapsulates // TODO
		{"L10 10L0 20zM20 0L10 10L20 20z", []bool{true, true}, ppath.NonZero}, // outside
		{"L10 10zM2 2L8 8z", []bool{true, true}, ppath.NonZero},               // zero-area overlap
		{"L10 10zM10 0L5 5L20 10z", []bool{true, true}, ppath.NonZero},        // outside

		// equal
		{"L10 -10L20 0L10 10zL10 -10L20 0L10 10z", []bool{true, true}, ppath.NonZero},
		{"L10 -10L20 0L10 10zA10 10 0 0 1 20 0A10 10 0 0 1 0 0z", []bool{true, true}, ppath.NonZero},
		//{"L10 -10L20 0L10 10zA10 10 0 0 0 20 0A10 10 0 0 0 0 0z", []bool{false, true}, NonZero}, // TODO
		//{"L10 -10L20 0L10 10zQ10 0 10 10Q10 0 20 0Q10 0 10 -10Q10 0 0 0z", []bool{true, false}, NonZero}, // TODO

		// open
		{"L10 10L0 20", []bool{true}, ppath.NonZero},
		{"L10 10L0 20M0 -5L0 5L-5 0z", []bool{true, true}, ppath.NonZero},
		{"L10 10L0 20M0 -5L0 5L5 0z", []bool{true, true}, ppath.NonZero},
	}
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			filling := Filling(ppath.MustParseSVGPath(tt.p), tt.rule)
			assert.Equal(t, filling, tt.filling)
		})
	}
}

func TestPathBounds(t *testing.T) {
	var tts = []struct {
		p      string
		bounds math32.Box2
	}{
		{"", math32.Box2{}},
		{"Q50 100 100 0", math32.B2(0, 0, 100, 50)},
		{"Q100 50 0 100", math32.B2(0, 0, 50, 100)},
		{"Q0 0 100 0", math32.B2(0, 0, 100, 0)},
		{"Q100 0 100 0", math32.B2(0, 0, 100, 0)},
		{"Q100 0 100 100", math32.B2(0, 0, 100, 100)},
		{"C0 0 100 0 100 0", math32.B2(0, 0, 100, 0)},
		{"C0 100 100 100 100 0", math32.B2(0, 0, 100, 75)},
		{"C0 0 100 90 100 0", math32.B2(0, 0, 100, 40)},
		{"C0 90 100 0 100 0", math32.B2(0, 0, 100, 40)},
		{"C100 100 0 100 100 0", math32.B2(0, 0, 100, 75)},
		{"C66.667 0 100 33.333 100 100", math32.B2(0, 0, 100, 100)},
		{"M3.1125 1.7812C3.4406 1.7812 3.5562 1.5938 3.4578 1.2656", math32.B2(3.1125, 1.2656, 3.1125+0.379252, 1.2656+0.515599)},
		{"A100 100 0 0 0 100 100", math32.B2(0, 0, 100, 100)},
		{"A50 100 90 0 0 200 0", math32.B2(0, 0, 200, 50)},
		{"A100 100 0 1 0 -100 100", math32.B2(-200, -100, 0, 100)}, // hit xmin, ymin
		{"A100 100 0 1 1 -100 100", math32.B2(-100, 0, 100, 200)},  // hit xmax, ymax
	}
	origEpsilon := ppath.Epsilon
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			ppath.Epsilon = origEpsilon
			bounds := Bounds(ppath.MustParseSVGPath(tt.p))
			ppath.Epsilon = 1e-6
			tolEqualVec2(t, bounds.Min, tt.bounds.Min)
		})
	}
	ppath.Epsilon = origEpsilon
}

// for quadratic Bézier use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D2*(1-t)*t*50.00+%2B+t%5E2*100.00,+y%3D2*(1-t)*t*66.67+%2B+t%5E2*0.00%7D+from+0+to+1
// for cubic Bézier use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D3*(1-t)%5E2*t*0.00+%2B+3*(1-t)*t%5E2*100.00+%2B+t%5E3*100.00,+y%3D3*(1-t)%5E2*t*66.67+%2B+3*(1-t)*t%5E2*66.67+%2B+t%5E3*0.00%7D+from+0+to+1
// for ellipse use https://www.wolframalpha.com/input/?i=length+of+the+curve+%7Bx%3D10.00*cos(t),+y%3D20.0*sin(t)%7D+from+0+to+pi
func TestPathLength(t *testing.T) {
	var tts = []struct {
		p      string
		length float32
	}{
		{"M10 0z", 0.0},
		{"Q50 66.67 100 0", 124.533},
		{"Q100 0 100 0", 100.0000},
		{"C0 66.67 100 66.67 100 0", 158.5864},
		{"C0 0 100 66.67 100 0", 125.746},
		{"C0 0 100 0 100 0", 100.0000},
		{"C100 66.67 0 66.67 100 0", 143.9746},
		{"A10 20 0 0 0 20 0", 48.4422},
		{"A10 20 0 0 1 20 0", 48.4422},
		{"A10 20 0 1 0 20 0", 48.4422},
		{"A10 20 0 1 1 20 0", 48.4422},
		{"A10 20 30 0 0 20 0", 31.4622},
	}
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			length := Length(ppath.MustParseSVGPath(tt.p))
			if tt.length == 0.0 {
				assert.True(t, length == 0)
			} else {
				lerr := math32.Abs(tt.length-length) / length
				assert.True(t, lerr < 0.01)
			}
		})
	}
}

func TestPathSplitAt(t *testing.T) {
	var tts = []struct {
		p  string
		d  []float32
		rs []string
	}{
		{"L4 3L8 0z", []float32{}, []string{"L4 3L8 0z"}},
		{"M2 0L4 3Q10 10 20 0C20 10 30 10 30 0A10 10 0 0 0 50 0z", []float32{0.0}, []string{"M2 0L4 3Q10 10 20 0C20 10 30 10 30 0A10 10 0 0 0 50 0L2 0"}},
		{"L4 3L8 0z", []float32{0.0, 5.0, 10.0, 18.0}, []string{"L4 3", "M4 3L8 0", "M8 0L0 0"}},
		{"L4 3L8 0z", []float32{5.0, 20.0}, []string{"L4 3", "M4 3L8 0L0 0"}},
		{"L4 3L8 0z", []float32{2.5, 7.5, 14.0}, []string{"L2 1.5", "M2 1.5L4 3L6 1.5", "M6 1.5L8 0L4 0", "M4 0L0 0"}},
		{"Q10 10 20 0", []float32{11.477858}, []string{"Q5 5 10 5", "M10 5Q15 5 20 0"}},
		{"C0 10 20 10 20 0", []float32{13.947108}, []string{"C0 5 5 7.5 10 7.5", "M10 7.5C15 7.5 20 5 20 0"}},
		// todo:
		// {"A10 10 0 0 1 -20 0", []float32{15.707963}, []string{"A10 10 0 0 1 -10 10", "M-10 10A10 10 0 0 1 -20 0"}},
		// {"A10 10 0 0 0 20 0", []float32{15.707963}, []string{"A10 10 0 0 0 10 10", "M10 10A10 10 0 0 0 20 0"}},
		// {"A10 10 0 1 0 2.9289 -7.0711", []float32{15.707963}, []string{"A10 10 0 0 0 10.024 9.9999", "M10.024 9.9999A10 10 0 1 0 2.9289 -7.0711"}},
	}
	origEpsilon := ppath.Epsilon
	for _, tt := range tts {
		t.Run(tt.p, func(t *testing.T) {
			ppath.Epsilon = origEpsilon
			p := ppath.MustParseSVGPath(tt.p)
			ps := SplitAt(p, tt.d...)
			ppath.Epsilon = 1e-3
			if len(ps) != len(tt.rs) {
				origs := []string{}
				for _, p := range ps {
					origs = append(origs, p.String())
				}
				assert.Equal(t, strings.Join(tt.rs, "\n"), strings.Join(origs, "\n"))
			} else {
				for i, p := range ps {
					tolassert.EqualTolSlice(t, ppath.MustParseSVGPath(tt.rs[i]), p, 1.0e-3)
				}
			}
		})
	}
	ppath.Epsilon = origEpsilon
}
