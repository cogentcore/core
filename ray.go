// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Ray represents an oriented 3D line segment defined by an origin point and a direction vector.
type Ray struct {
	Origin Vec3
	Dir    Vec3
}

// NewRay creates and returns a pointer to a Ray object with
// the specified origin and direction vectors.
// If a nil pointer is supplied for any of the parameters,
// the zero vector will be used.
func NewRay(origin, dir Vec3) *Ray {
	return &Ray{origin, dir}
}

// Set sets the origin and direction vectors of this Ray.
func (ray *Ray) Set(origin, dir Vec3) {
	ray.Origin = origin
	ray.Dir = dir
}

// At calculates the point in the ray which is at the specified t distance from the origin
// along its direction.
func (ray *Ray) At(t float32) Vec3 {
	return ray.Dir.MulScalar(t).Add(ray.Origin)
}

// Recast sets the new origin of the ray at the specified distance t
// from its origin along its direction.
func (ray *Ray) Recast(t float32) {
	ray.Origin = ray.At(t)
}

// ClosestPointToPoint calculates the point in the ray which is closest to the specified point.
func (ray *Ray) ClosestPointToPoint(point Vec3) Vec3 {
	dirDist := point.Sub(ray.Origin).Dot(ray.Dir)
	if dirDist < 0 {
		return ray.Origin
	}
	return ray.Dir.MulScalar(dirDist).Add(ray.Origin)
}

// DistToPoint returns the smallest distance
// from the ray direction vector to the specified point.
func (ray *Ray) DistToPoint(point Vec3) float32 {
	return Sqrt(ray.DistSqToPoint(point))
}

// DistSqToPoint returns the smallest squared distance
// from the ray direction vector to the specified point.
// If the ray was pointed directly at the point this distance would be 0.
func (ray *Ray) DistSqToPoint(point Vec3) float32 {
	dirDist := point.Sub(ray.Origin).Dot(ray.Dir)
	// point behind the ray
	if dirDist < 0 {
		return ray.Origin.DistTo(point)
	}
	return ray.Dir.MulScalar(dirDist).Add(ray.Origin).DistToSquared(point)
}

// DistSqToSegment returns the smallest squared distance
// from this ray to the line segment from v0 to v1.
// If optPointOnRay Vec3 is not nil,
// it is set with the coordinates of the point on the ray.
// if optPointOnSegment Vec3 is not nil,
// it is set with the coordinates of the point on the segment.
func (ray *Ray) DistSqToSegment(v0, v1 Vec3, optPointOnRay, optPointOnSegment *Vec3) float32 {
	segCenter := v0.Add(v1).MulScalar(0.5)
	segDir := v1.Sub(v0).Normal()
	diff := ray.Origin.Sub(segCenter)

	segExtent := v0.DistTo(v1) * 0.5
	a01 := -ray.Dir.Dot(segDir)
	b0 := diff.Dot(ray.Dir)
	b1 := -diff.Dot(segDir)
	c := diff.LengthSq()
	det := Abs(1 - a01*a01)

	var s0, s1, sqrDist, extDet float32
	if det > 0 {
		// The ray and segment are not parallel.
		s0 = a01*b1 - b0
		s1 = a01*b0 - b1
		extDet = segExtent * det

		if s0 >= 0 {
			if s1 >= -extDet {
				if s1 <= extDet {
					// region 0
					// Minimum at interior points of ray and segment.
					invDet := 1 / det
					s0 *= invDet
					s1 *= invDet
					sqrDist = s0*(s0+a01*s1+2*b0) + s1*(a01*s0+s1+2*b1) + c
				} else {
					// region 1
					s1 = segExtent
					s0 = Max(0, -(a01*s1 + b0))
					sqrDist = -s0*s0 + s1*(s1+2*b1) + c
				}
			} else {
				// region 5
				s1 = -segExtent
				s0 = Max(0, -(a01*s1 + b0))
				sqrDist = -s0*s0 + s1*(s1+2*b1) + c
			}
		} else {
			if s1 <= -extDet {
				// region 4
				s0 = Max(0, -(-a01*segExtent + b0))
				if s0 > 0 {
					s1 = -segExtent
				} else {
					s1 = Min(Max(-segExtent, -b1), segExtent)
				}
				sqrDist = -s0*s0 + s1*(s1+2*b1) + c
			} else if s1 <= extDet {
				// region 3
				s0 = 0
				s1 = Min(Max(-segExtent, -b1), segExtent)
				sqrDist = s1*(s1+2*b1) + c

			} else {
				// region 2
				s0 = Max(0, -(a01*segExtent + b0))
				if s0 > 0 {
					s1 = segExtent
				} else {
					s1 = Min(Max(-segExtent, -b1), segExtent)
				}
				sqrDist = -s0*s0 + s1*(s1+2*b1) + c
			}
		}
	} else {
		// Ray and segment are parallel.
		if a01 > 0 {
			s1 = -segExtent
		} else {
			s1 = segExtent
		}
		s0 = Max(0, -(a01*s1 + b0))
		sqrDist = -s0*s0 + s1*(s1+2*b1) + c
	}

	if optPointOnRay != nil {
		*optPointOnRay = ray.Dir.MulScalar(s0).Add(ray.Origin)
	}

	if optPointOnSegment != nil {
		*optPointOnSegment = segDir.MulScalar(s1).Add(segCenter)
	}
	return sqrDist
}

// IsIntersectionSphere returns if this ray intersects with the specified sphere.
func (ray *Ray) IsIntersectionSphere(sphere Sphere) bool {
	if ray.DistToPoint(sphere.Center) <= sphere.Radius {
		return true
	}
	return false
}

// IntersectSphere calculates the point which is the intersection of this ray with the specified sphere.
// If no intersection is found false is returne.
func (ray *Ray) IntersectSphere(sphere Sphere) (Vec3, bool) {
	v1 := sphere.Center.Sub(ray.Origin)
	tca := v1.Dot(ray.Dir)
	d2 := v1.Dot(v1) - tca*tca
	radius2 := sphere.Radius * sphere.Radius

	if d2 > radius2 {
		return v1, false
	}
	thc := Sqrt(radius2 - d2)
	// t0 = first intersect point - entrance on front of sphere
	t0 := tca - thc
	// t1 = second intersect point - exit point on back of sphere
	t1 := tca + thc
	// test to see if both t0 and t1 are behind the ray - if so, return null
	if t0 < 0 && t1 < 0 {
		return v1, false
	}

	// test to see if t0 is behind the ray:
	// if it is, the ray is inside the sphere, so return the second exit point scaled by t1,
	// in order to always return an intersect point that is in front of the ray.
	if t0 < 0 {
		return ray.At(t1), true
	}
	// else t0 is in front of the ray, so return the first collision point scaled by t0
	return ray.At(t0), true
}

// IsIntersectPlane returns if this ray intersects the specified plane.
func (ray *Ray) IsIntersectPlane(plane Plane) bool {
	distToPoint := plane.DistToPoint(ray.Origin)
	if distToPoint == 0 {
		return true
	}
	denom := plane.Norm.Dot(ray.Dir)
	if denom*distToPoint < 0 {
		return true
	}
	// ray origin is behind the plane (and is pointing behind it)
	return false
}

// DistToPlane returns the distance of this ray origin to its intersection point in the plane.
// If the ray does not intersects the plane, returns NaN.
func (ray *Ray) DistToPlane(plane Plane) float32 {
	denom := plane.Norm.Dot(ray.Dir)
	if denom == 0 {
		// line is coplanar, return origin
		if plane.DistToPoint(ray.Origin) == 0 {
			return 0
		}
		return NaN()
	}
	t := -(ray.Origin.Dot(plane.Norm) + plane.Off) / denom
	// Return if the ray never intersects the plane
	if t >= 0 {
		return t
	}
	return NaN()
}

// IntersectPlane calculates the point which is the intersection of this ray with the specified plane.
// If no intersection is found false is returned.
func (ray *Ray) IntersectPlane(plane Plane) (Vec3, bool) {
	t := ray.DistToPlane(plane)
	if t == NaN() {
		return ray.Origin, false
	}
	return ray.At(t), true
}

// IntersectsBox returns if this ray intersects the specified box.
func (ray *Ray) IntersectsBox(box Box3) bool {
	_, yes := ray.IntersectBox(box)
	return yes
}

// IntersectBox calculates the point which is the intersection of this ray with the specified box.
// If no intersection is found false is returned.
func (ray *Ray) IntersectBox(box Box3) (Vec3, bool) {
	// http://www.scratchapixel.com/lessons/3d-basic-lessons/lesson-7-intersecting-simple-shapes/ray-box-intersection/

	var tmin, tmax, tymin, tymax, tzmin, tzmax float32

	invdirx := 1 / ray.Dir.X
	invdiry := 1 / ray.Dir.Y
	invdirz := 1 / ray.Dir.Z

	var origin = ray.Origin

	if invdirx >= 0 {
		tmin = (box.Min.X - origin.X) * invdirx
		tmax = (box.Max.X - origin.X) * invdirx
	} else {
		tmin = (box.Max.X - origin.X) * invdirx
		tmax = (box.Min.X - origin.X) * invdirx
	}

	if invdiry >= 0 {
		tymin = (box.Min.Y - origin.Y) * invdiry
		tymax = (box.Max.Y - origin.Y) * invdiry
	} else {
		tymin = (box.Max.Y - origin.Y) * invdiry
		tymax = (box.Min.Y - origin.Y) * invdiry
	}

	if (tmin > tymax) || (tymin > tmax) {
		return ray.Origin, false
	}

	// These lines also handle the case where tmin or tmax is NaN
	// (result of 0 * Infinity). x !== x returns true if x is NaN

	if tymin > tmin || tmin != tmin {
		tmin = tymin
	}

	if tymax < tmax || tmax != tmax {
		tmax = tymax
	}

	if invdirz >= 0 {
		tzmin = (box.Min.Z - origin.Z) * invdirz
		tzmax = (box.Max.Z - origin.Z) * invdirz
	} else {
		tzmin = (box.Max.Z - origin.Z) * invdirz
		tzmax = (box.Min.Z - origin.Z) * invdirz
	}

	if (tmin > tzmax) || (tzmin > tmax) {
		return ray.Origin, false
	}

	if tzmin > tmin || tmin != tmin {
		tmin = tzmin
	}

	if tzmax < tmax || tmax != tmax {
		tmax = tzmax
	}

	//return point closest to the ray (positive side)

	if tmax < 0 {
		return ray.Origin, false
	}

	if tmin >= 0 {
		return ray.At(tmin), true
	}
	return ray.At(tmax), true
}

// IntersectTriangle returns if this ray intersects the triangle with the face
// defined by points a, b, c. Returns true if it intersects and the point
// parameter with the intersected point coordinates.
// If backfaceCulling is false it ignores the intersection if the face is not oriented
// in the ray direction.
func (ray *Ray) IntersectTriangle(a, b, c Vec3, backfaceCulling bool) (Vec3, bool) {
	edge1 := b.Sub(a)
	edge2 := c.Sub(a)
	normal := edge1.Cross(edge2)

	// Solve Q + t*D = b1*E1 + b2*E2 (Q = kDiff, D = ray direction,
	// E1 = kEdge1, E2 = kEdge2, N = Cross(E1,E2)) by
	//   |Dot(D,N)|*b1 = sign(Dot(D,N))*Dot(D,Cross(Q,E2))
	//   |Dot(D,N)|*b2 = sign(Dot(D,N))*Dot(D,Cross(E1,Q))
	//   |Dot(D,N)|*t = -sign(Dot(D,N))*Dot(Q,N)
	DdN := ray.Dir.Dot(normal)
	var sign float32

	if DdN > 0 {
		if backfaceCulling {
			return ray.Origin, false
		}
		sign = 1
	} else if DdN < 0 {
		sign = -1
		DdN = -DdN
	} else {
		return ray.Origin, false
	}

	diff := ray.Origin.Sub(a)
	DdQxE2 := sign * ray.Dir.Dot(diff.Cross(edge2))

	// b1 < 0, no intersection
	if DdQxE2 < 0 {
		return ray.Origin, false
	}

	DdE1xQ := sign * ray.Dir.Dot(edge1.Cross(diff))
	// b2 < 0, no intersection
	if DdE1xQ < 0 {
		return ray.Origin, false
	}

	// b1+b2 > 1, no intersection
	if DdQxE2+DdE1xQ > DdN {
		return ray.Origin, false
	}

	// Line intersects triangle, check if ray does.
	QdN := -sign * diff.Dot(normal)

	// t < 0, no intersection
	if QdN < 0 {
		return ray.Origin, false
	}

	// Ray intersects triangle.
	return ray.At(QdN / DdN), true
}

// MulMat4 multiplies this ray origin and direction
// by the specified matrix4, basically transforming this ray coordinates.
func (ray *Ray) ApplyMat4(mat4 *Mat4) {
	ray.Dir = ray.Dir.Add(ray.Origin).MulMat4(mat4)
	ray.Origin = ray.Origin.MulMat4(mat4)
	ray.Dir.SetSub(ray.Origin)
	ray.Dir.SetNormal()
}

// IsEqual returns if this ray is equal to other
func (ray *Ray) IsEqual(other Ray) bool {
	return ray.Origin.IsEqual(other.Origin) && ray.Dir.IsEqual(other.Dir)
}
