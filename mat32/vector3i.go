// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Vec3i is a 3D vector/point with X, Y and Z int32 components.
type Vec3i struct {
	X int32
	Y int32
	Z int32
}

// NewVec3i creates and returns a pointer to a new Vec3i with
// the specified x, y and y components
func NewVec3i(x, y, z int32) *Vec3i {
	return &Vec3i{X: x, Y: y, Z: z}
}

// Set sets this vector X, Y and Z components.
// Returns the pointer to this updated vector.
func (v *Vec3i) Set(x, y, z int32) *Vec3i {
	v.X = x
	v.Y = y
	v.Z = z
	return v
}

// SetX sets this vector X component.
// Returns the pointer to this updated Vector.
func (v *Vec3i) SetX(x int32) *Vec3i {
	v.X = x
	return v
}

// SetY sets this vector Y component.
// Returns the pointer to this updated vector.
func (v *Vec3i) SetY(y int32) *Vec3i {
	v.Y = y
	return v
}

// SetZ sets this vector Z component.
// Returns the pointer to this updated vector.
func (v *Vec3i) SetZ(z int32) *Vec3i {
	v.Z = z
	return v
}

// SetComponent sets this vector component value by component index.
// Returns the pointer to this updated vector
func (v *Vec3i) SetComponent(comp Components, value int32) {
	switch comp {
	case X:
		v.X = value
	case Y:
		v.Y = value
	case Z:
		v.Z = value
	default:
		panic("index is out of range: ")
	}
}

// Component returns this vector component
func (v *Vec3i) Component(comp Components) int32 {
	switch comp {
	case X:
		return v.X
	case Y:
		return v.Y
	case Z:
		return v.Z
	default:
		panic("index is out of range")
	}
}

// SetByName sets this vector component value by its case insensitive name: "x", "y", or "z".
func (v *Vec3i) SetByName(name string, value int32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	case "z", "Z":
		v.Z = value
	default:
		panic("Invalid Vec3i component name: " + name)
	}
}

// Zero sets this vector X, Y and Z components to be zero.
// Returns the pointer to this updated vector.
func (v *Vec3i) Zero() *Vec3i {
	v.X = 0
	v.Y = 0
	v.Z = 0
	return v
}

// Copy copies other vector to this one.
// It is equivalent to: *v = *other.
// Returns the pointer to this updated vector.
func (v *Vec3i) Copy(other *Vec3i) *Vec3i {
	*v = *other
	return v
}

// Add adds other vector to this one.
// Returns the pointer to this updated vector.
func (v *Vec3i) Add(other *Vec3i) *Vec3i {
	v.X += other.X
	v.Y += other.Y
	v.Z += other.Z
	return v
}

// AddScalar adds scalar s to each component of this vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) AddScalar(s int32) *Vec3i {
	v.X += s
	v.Y += s
	v.Z += s
	return v
}

// AddVectors adds vectors a and b to this one.
// Returns the pointer to this updated vector.
func (v *Vec3i) AddVectors(a, b *Vec3i) *Vec3i {
	v.X = a.X + b.X
	v.Y = a.Y + b.Y
	v.Z = a.Z + b.Z
	return v
}

// Sub subtracts other vector from this one.
// Returns the pointer to this updated vector.
func (v *Vec3i) Sub(other *Vec3i) *Vec3i {
	v.X -= other.X
	v.Y -= other.Y
	v.Z -= other.Z
	return v
}

// SubScalar subtracts scalar s from each component of this vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) SubScalar(s int32) *Vec3i {
	v.X -= s
	v.Y -= s
	v.Z -= s
	return v
}

// SubVectors sets this vector to a - b.
// Returns the pointer to this updated vector.
func (v *Vec3i) SubVectors(a, b *Vec3i) *Vec3i {
	v.X = a.X - b.X
	v.Y = a.Y - b.Y
	v.Z = a.Z - b.Z
	return v
}

// Multiply multiplies each component of this vector by the corresponding one from other vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) Multiply(other *Vec3i) *Vec3i {
	v.X *= other.X
	v.Y *= other.Y
	v.Z *= other.Z
	return v
}

// MultiplyScalar multiplies each component of this vector by the scalar s.
// Returns the pointer to this updated vector.
func (v *Vec3i) MultiplyScalar(s int32) *Vec3i {
	v.X *= s
	v.Y *= s
	v.Z *= s
	return v
}

// Divide divides each component of this vector by the corresponding one from other vector.
// Returns the pointer to this updated vector
func (v *Vec3i) Divide(other *Vec3i) *Vec3i {
	v.X /= other.X
	v.Y /= other.Y
	v.Z /= other.Z
	return v
}

// DivideScalar divides each component of this vector by the scalar s.
// If scalar is zero, sets this vector to zero.
// Returns the pointer to this updated vector.
func (v *Vec3i) DivideScalar(scalar int32) *Vec3i {
	if scalar != 0 {
		invScalar := 1 / scalar
		v.X *= invScalar
		v.Y *= invScalar
		v.Z *= invScalar
	} else {
		v.X = 0
		v.Y = 0
		v.Z = 0
	}
	return v
}

// Min sets this vector components to the minimum values of itself and other vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) Min(other *Vec3i) *Vec3i {
	if v.X > other.X {
		v.X = other.X
	}
	if v.Y > other.Y {
		v.Y = other.Y
	}
	if v.Z > other.Z {
		v.Z = other.Z
	}
	return v
}

// Max sets this vector components to the maximum value of itself and other vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) Max(other *Vec3i) *Vec3i {
	if v.X < other.X {
		v.X = other.X
	}
	if v.Y < other.Y {
		v.Y = other.Y
	}
	if v.Z < other.Z {
		v.Z = other.Z
	}
	return v
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding component of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
// Returns the pointer to this updated vector.
func (v *Vec3i) Clamp(min, max *Vec3i) *Vec3i {
	if v.X < min.X {
		v.X = min.X
	} else if v.X > max.X {
		v.X = max.X
	}

	if v.Y < min.Y {
		v.Y = min.Y
	} else if v.Y > max.Y {
		v.Y = max.Y
	}

	if v.Z < min.Z {
		v.Z = min.Z
	} else if v.Z > max.Z {
		v.Z = max.Z
	}
	return v
}

// ClampScalar sets this vector components to be no less than minVal and not greater than maxVal.
// Returns the pointer to this updated vector.
func (v *Vec3i) ClampScalar(minVal, maxVal int32) *Vec3i {
	min := NewVec3i(minVal, minVal, minVal)
	max := NewVec3i(maxVal, maxVal, maxVal)
	return v.Clamp(min, max)
}

// Negate negates each of this vector's components.
// Returns the pointer to this updated vector.
func (v *Vec3i) Negate() *Vec3i {
	v.X = -v.X
	v.Y = -v.Y
	v.Z = -v.Z
	return v
}

// Equals returns if this vector is equal to other.
func (v *Vec3i) Equals(other *Vec3i) bool {
	return (other.X == v.X) && (other.Y == v.Y) && (other.Z == v.Z)
}

// FromArray sets this vector's components from the specified array and offset
// Returns the pointer to this updated vector.
func (v *Vec3i) FromArray(array []int32, offset int) *Vec3i {
	v.X = array[offset]
	v.Y = array[offset+1]
	v.Z = array[offset+2]
	return v
}

// ToArray copies this vector's components to array starting at offset.
// Returns the array.
func (v *Vec3i) ToArray(array []int32, offset int) []int32 {
	array[offset] = v.X
	array[offset+1] = v.Y
	array[offset+2] = v.Z
	return array
}

// MultiplyVectors multiply vectors a and b storing the result in this vector.
// Returns the pointer to this updated vector.
func (v *Vec3i) MultiplyVectors(a, b *Vec3i) *Vec3i {
	v.X = a.X * b.X
	v.Y = a.Y * b.Y
	v.Z = a.Z * b.Z
	return v
}

// Clone returns a copy of this vector
func (v *Vec3i) Clone() *Vec3i {
	return NewVec3i(v.X, v.Y, v.Z)
}
