// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPathStroke(t *testing.T) {
	tolerance := float32(1.0)
	var tts = []struct {
		orig   string
		w      float32
		cp     Capper
		jr     Joiner
		stroke string
	}{
		{"M10 10", 2.0, RoundCap, RoundJoin, ""},
		{"M10 10z", 2.0, RoundCap, RoundJoin, ""},
		//{"M10 10L10 5", 2.0, RoundCap, RoundJoin, "M9 5A1 1 0 0 1 11 5L11 10A1 1 0 0 1 9 10z"},
		{"M10 10L10 5", 2.0, ButtCap, RoundJoin, "M9 5L11 5L11 10L9 10z"},
		{"M10 10L10 5", 2.0, SquareCap, RoundJoin, "M9 4L11 4L11 5L11 10L11 11L9 11z"},

		{"L10 0L20 0", 2.0, ButtCap, RoundJoin, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		//{"L10 0L10 10", 2.0, ButtCap, RoundJoin, "M9 1L0 1L0 -1L10 -1A1 1 0 0 1 11 0L11 10L9 10z"},
		//{"L10 0L10 -10", 2.0, ButtCap, RoundJoin, "M9 -1L9 -10L11 -10L11 0A1 1 0 0 1 10 1L0 1L0 -1z"},

		{"L10 0L20 0", 2.0, ButtCap, BevelJoin, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"L10 0L10 10", 2.0, ButtCap, BevelJoin, "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"L10 0L10 -10", 2.0, ButtCap, BevelJoin, "M0 -1L9 -1L9 -10L11 -10L11 0L10 1L0 1z"},

		{"L10 0L20 0", 2.0, ButtCap, MiterJoiner{BevelJoin, 2.0}, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"L10 0L5 0", 2.0, ButtCap, MiterJoiner{BevelJoin, 2.0}, "M0 -1L10 -1L10 1L0 1z"},
		{"L10 0L10 10", 2.0, ButtCap, MiterJoiner{BevelJoin, 1.0}, "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"L10 0L10 10", 2.0, ButtCap, MiterJoiner{BevelJoin, 2.0}, "M0 -1L10 -1L11 -1L11 0L11 10L9 10L9 1L0 1z"},
		{"L10 0L10 -10", 2.0, ButtCap, MiterJoiner{BevelJoin, 2.0}, "M0 -1L9 -1L9 -10L11 -10L11 0L11 1L10 1L0 1z"},

		{"L10 0L20 0", 2.0, ButtCap, ArcsJoiner{BevelJoin, 2.0}, "M0 -1L10 -1L20 -1L20 1L10 1L0 1z"},
		{"L10 0L5 0", 2.0, ButtCap, ArcsJoiner{BevelJoin, 2.0}, "M0 -1L10 -1L10 1L0 1z"},
		{"L10 0L10 10", 2.0, ButtCap, ArcsJoiner{BevelJoin, 1.0}, "M0 -1L10 -1L11 0L11 10L9 10L9 1L0 1z"},

		{"L10 0L10 10L0 10z", 2.0, ButtCap, MiterJoin, "M-1 -1L11 -1L11 11L-1 11zM1 1L1 9L9 9L9 1z"},
		{"L10 0L10 10L0 10z", 2.0, ButtCap, BevelJoin, "M-1 0L0 -1L10 -1L11 0L11 10L10 11L0 11L-1 10zM1 1L1 9L9 9L9 1z"},
		{"L0 10L10 10L10 0z", 2.0, ButtCap, BevelJoin, "M-1 0L0 -1L10 -1L11 0L11 10L10 11L0 11L-1 10zM1 1L1 9L9 9L9 1z"},
		{"Q10 0 10 10", 2.0, ButtCap, BevelJoin, "M0 -1L9.5137 3.4975L11 10L9 10L7.7845 4.5025L0 1z"},
		{"C0 10 10 10 10 0", 2.0, ButtCap, BevelJoin, "M-1 0L1 0L3.5291 6.0900L7.4502 5.2589L9 0L11 0L8.9701 6.5589L2.5234 7.8188z"},
		//{"A10 5 0 0 0 20 0", 2.0, ButtCap, BevelJoin, "M1 0A9 4 0 0 0 19 0L21 0A11 6 0 0 1 -1 0z"}, // TODO: enable tests for ellipses when Settle supports them
		//{"A10 5 0 0 1 20 0", 2.0, ButtCap, BevelJoin, "M-1 0A11 6 0 0 1 21 0L19 0A9 4 0 0 0 1 0z"},
		//{"M5 2L2 2A2 2 0 0 0 0 0", 2.0, ButtCap, BevelJoin, "M2.8284 1L5 1L5 3L2 3L1 2A1 1 0 0 0 0 1L0 -1A3 3 0 0 1 2.8284 1z"},

		// two circle quadrants joining at 90 degrees
		//{"A10 10 0 0 1 10 10A10 10 0 0 1 0 0z", 2.0, ButtCap, ArcsJoin, "M0 -1A11 11 0 0 1 11 10A11 11 0 0 1 10.958 10.958A11 11 0 0 1 10 11A11 11 0 0 1 -1 0A11 11 0 0 1 -0.958 -0.958A11 11 0 0 1 0 -1zM1.06230 1.06230A9 9 0 0 0 8.9370 8.9370A9 9 0 0 0 1.06230 1.0630z"},

		// circles joining at one point (10,0), stroke will never join
		//{"A5 5 0 0 0 10 0A10 10 0 0 1 0 10", 2.0, ButtCap, ArcsJoin, "M7 5.6569A6 6 0 0 1 -1 0L1 0A4 4 0 0 0 9 0L11 0A11 11 0 0 1 0 11L0 9A9 9 0 0 0 7 5.6569z"},

		// circle and line intersecting in one point
		//{"A2 2 0 0 1 2 2L5 2", 2.0, ButtCap, ArcsJoiner{BevelJoin, 10.0}, "M2.8284 1L5 1L5 3L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L0 -1A3 3 0 0 1 2.8284 1z"},
		//{"M0 4A2 2 0 0 0 2 2L5 2", 2.0, ButtCap, ArcsJoiner{BevelJoin, 10.0}, "M2.8284 3A3 3 0 0 1 0 5L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L5 1L5 3z"},
		//{"M5 2L2 2A2 2 0 0 0 0 0", 2.0, ButtCap, ArcsJoiner{BevelJoin, 10.0}, "M2.8284 1L5 1L5 3L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L0 -1A3 3 0 0 1 2.8284 1z"},
		//{"M5 2L2 2A2 2 0 0 1 0 4", 2.0, ButtCap, ArcsJoiner{BevelJoin, 10.0}, "M2.8284 3A3 3 0 0 1 0 5L0 3A1 1 0 0 0 1 2A1 1 0 0 0 0 1L5 1L5 3z"},

		// cut by limit
		//{"A2 2 0 0 1 2 2L5 2", 2.0, ButtCap, ArcsJoiner{BevelJoin, 1.0}, "M2.8284 1L5 1L5 3L2 3L1 2A1 1 0 0 0 0 1L0 -1A3 3 0 0 1 2.8284 1z"},

		// no intersection
		//{"A2 2 0 0 1 2 2L5 2", 3.0, ButtCap, ArcsJoiner{BevelJoin, 10.0}, "M3.1623 0.5L5 0.5L5 3.5L2 3.5L0.5 2A0.5 0.5 0 0 0 0 1.5L0 -1.5A3.5 3.5 0 0 1 3.1623 0.5z"},
	}
	origEpsilon := Epsilon
	for _, tt := range tts {
		t.Run(tt.orig, func(t *testing.T) {
			Epsilon = origEpsilon
			stroke := MustParseSVGPath(tt.orig).Stroke(tt.w, tt.cp, tt.jr, tolerance)
			Epsilon = 1e-3
			assert.InDeltaSlice(t, MustParseSVGPath(tt.stroke), stroke, 1.0e-4)
		})
	}
	Epsilon = origEpsilon
}

func TestPathStrokeEllipse(t *testing.T) {
	rx, ry := float32(20.0), float32(10.0)
	nphi := 12
	ntheta := 120
	for iphi := 0; iphi < nphi; iphi++ {
		phi := float32(iphi) / float32(nphi) * math.Pi
		for itheta := 0; itheta < ntheta; itheta++ {
			theta := float32(itheta) / float32(ntheta) * 2.0 * math.Pi
			outer := EllipsePos(rx+1.0, ry+1.0, phi, 0.0, 0.0, theta)
			inner := EllipsePos(rx-1.0, ry-1.0, phi, 0.0, 0.0, theta)
			assert.InDelta(t, float32(2.0), outer.Sub(inner).Length(), 1.0e-4, fmt.Sprintf("phi=%g theta=%g", phi, theta))
		}
	}
}

func TestPathOffset(t *testing.T) {
	tolerance := float32(0.01)
	var tts = []struct {
		orig   string
		w      float32
		offset string
	}{
		{"L10 0L10 10L0 10z", 0.0, "L10 0L10 10L0 10z"},
		//{"L10 0L10 10L0 10", 1.0, "M0 -1L10 -1A1 1 0 0 1 11 0L11 10A1 1 0 0 1 10 11L0 11"},
		//{"L10 0L10 10L0 10z", 1.0, "M10 -1A1 1 0 0 1 11 0L11 10A1 1 0 0 1 10 11L0 11A1 1 0 0 1 -1 10L-1 0A1 1 0 0 1 0 -1z"},
		{"L10 0L10 10L0 10z", -1.0, "M1 1L9 1L9 9L1 9z"},
		{"L10 0L5 0z", -1.0, "M-0.99268263 -0.18098975L-0.99268263 0.18098975L-0.86493423 0.51967767L-0.62587738 0.79148822L-0.30627632 0.9614421L0 1L10 1L10.30627632 0.9614421L10.62587738 0.79148822L10.86493423 0.51967767L10.992682630000001 0.18098975L10.992682630000001 -0.18098975L10.86493423 -0.51967767L10.62587738 -0.79148822L10.30627632 -0.9614421L10 -1L0 -1L-0.30627632 -0.9614421L-0.62587738 -0.79148822L-0.86493423 -0.51967767z"},
	}
	for _, tt := range tts {
		t.Run(fmt.Sprintf("%v/%v", tt.orig, tt.w), func(t *testing.T) {
			offset := MustParseSVGPath(tt.orig).Offset(tt.w, tolerance)
			assert.InDeltaSlice(t, MustParseSVGPath(tt.offset), offset, 1.0e-5)
		})
	}
}
