// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package math32

import (
	"testing"

	"cogentcore.org/core/base/tolassert"
	"github.com/chewxy/math32"
)

func TestLine2(t *testing.T) {
	st := Vec2(6, 12)
	ed := Vec2(12, 24)
	l := NewLine2(st, ed)
	ctr := l.Center()

	tolAssertEqualVector(t, Vec2(9, 18), ctr)
	tolAssertEqualVector(t, Vec2(6, 12), l.Delta())
	tolassert.EqualTol(t, 180, l.LengthSquared(), standardTol)
	tolassert.EqualTol(t, math32.Sqrt(180), l.Length(), standardTol)
	tolAssertEqualVector(t, st, l.ClosestPointToPoint(st))
	tolAssertEqualVector(t, ed, l.ClosestPointToPoint(ed))
	tolAssertEqualVector(t, ctr, l.ClosestPointToPoint(ctr))
	tolAssertEqualVector(t, st, l.ClosestPointToPoint(st.Sub(Vec2(2, 2))))
	tolAssertEqualVector(t, ed, l.ClosestPointToPoint(ed.Add(Vec2(2, 2))))
	tolAssertEqualVector(t, Vec2(7.8, 15.6), l.ClosestPointToPoint(st.Add(Vec2(3, 3))))
}
