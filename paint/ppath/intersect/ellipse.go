// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package intersect

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
)

func ellipseDeriv2(rx, ry, phi float32, theta float32) math32.Vector2 {
	sintheta, costheta := math32.Sincos(theta)
	sinphi, cosphi := math32.Sincos(phi)
	ddx := -rx*costheta*cosphi + ry*sintheta*sinphi
	ddy := -rx*costheta*sinphi - ry*sintheta*cosphi
	return math32.Vector2{ddx, ddy}
}

// ellipseLength calculates the length of the elliptical arc
// it uses Gauss-Legendre (n=5) and has an error of ~1% or less (empirical)
func ellipseLength(rx, ry, theta1, theta2 float32) float32 {
	if theta2 < theta1 {
		theta1, theta2 = theta2, theta1
	}
	speed := func(theta float32) float32 {
		return ppath.EllipseDeriv(rx, ry, 0.0, true, theta).Length()
	}
	return gaussLegendre5(speed, theta1, theta2)
}

func EllipseCurvatureRadius(rx, ry float32, sweep bool, theta float32) float32 {
	// positive for ccw / sweep
	// phi has no influence on the curvature
	dp := ppath.EllipseDeriv(rx, ry, 0.0, sweep, theta)
	ddp := ellipseDeriv2(rx, ry, 0.0, theta)
	a := dp.Cross(ddp)
	if ppath.Equal(a, 0.0) {
		return math32.NaN()
	}
	return math32.Pow(dp.X*dp.X+dp.Y*dp.Y, 1.5) / a
}

// ellipseSplit returns the new mid point, the two large parameters and the ok bool, the rest stays the same
func ellipseSplit(rx, ry, phi, cx, cy, theta0, theta1, theta float32) (math32.Vector2, bool, bool, bool) {
	if !ppath.IsAngleBetween(theta, theta0, theta1) {
		return math32.Vector2{}, false, false, false
	}

	mid := ppath.EllipsePos(rx, ry, phi, cx, cy, theta)
	large0, large1 := false, false
	if math32.Abs(theta-theta0) > math32.Pi {
		large0 = true
	} else if math32.Abs(theta-theta1) > math32.Pi {
		large1 = true
	}
	return mid, large0, large1, true
}

func xmonotoneEllipticArc(start math32.Vector2, rx, ry, phi float32, large, sweep bool, end math32.Vector2) ppath.Path {
	sign := float32(1.0)
	if !sweep {
		sign = -1.0
	}

	cx, cy, theta0, theta1 := ppath.EllipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
	sinphi, cosphi := math32.Sincos(phi)
	thetaRight := math32.Atan2(-ry*sinphi, rx*cosphi)
	thetaLeft := thetaRight + math32.Pi

	p := ppath.Path{}
	p.MoveTo(start.X, start.Y)
	left := !ppath.AngleEqual(thetaLeft, theta0) && ppath.AngleNorm(sign*(thetaLeft-theta0)) < ppath.AngleNorm(sign*(thetaRight-theta0))
	for t := theta0; !ppath.AngleEqual(t, theta1); {
		dt := ppath.AngleNorm(sign * (theta1 - t))
		if left {
			dt = math32.Min(dt, ppath.AngleNorm(sign*(thetaLeft-t)))
		} else {
			dt = math32.Min(dt, ppath.AngleNorm(sign*(thetaRight-t)))
		}
		t += sign * dt

		pos := ppath.EllipsePos(rx, ry, phi, cx, cy, t)
		p.ArcTo(rx, ry, phi, false, sweep, pos.X, pos.Y)
		left = !left
	}
	return p
}
