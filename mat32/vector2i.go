// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Initially copied from G3N: github.com/g3n/engine/math32
// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
// with modifications needed to suit GoGi functionality.

package mat32

// Vector2i is a 2D vector/point with X and Y integer32 components
type Vector2i struct {
	X int32
	Y int32
}

// NewVector2i creates and returns a pointer to a new Vector2i with
// the specified x and y components
func NewVector2i(x, y int32) *Vector2i {
	return &Vector2i{X: x, Y: y}
}

// NewVec2i creates and returns a pointer to a new zero-ed Vector2i.
func NewVec2i() *Vector2i {
	return &Vector2i{X: 0, Y: 0}
}

// Set sets this vector X and Y components.
// Returns the pointer to this updated vector.
func (v *Vector2i) Set(x, y int32) *Vector2i {
	v.X = x
	v.Y = y
	return v
}

// SetX sets this vector X component.
// Returns the pointer to this updated Vector.
func (v *Vector2i) SetX(x int32) *Vector2i {
	v.X = x
	return v
}

// SetY sets this vector Y component.
// Returns the pointer to this updated vector.
func (v *Vector2i) SetY(y int32) *Vector2i {
	v.Y = y
	return v
}

// SetComponent sets this vector component value by its index: 0 for X, 1 for Y.
// Returns the pointer to this updated vector
func (v *Vector2i) SetComponent(index int, value int32) *Vector2i {
	switch index {
	case 0:
		v.X = value
	case 1:
		v.Y = value
	default:
		panic("index is out of range")
	}
	return v
}

// Component returns this vector component by its index: 0 for X, 1 for Y
func (v *Vector2i) Component(index int) int32 {
	switch index {
	case 0:
		return v.X
	case 1:
		return v.Y
	default:
		panic("index is out of range")
	}
}

// SetByName sets this vector component value by its case insensitive name: "x" or "y".
func (v *Vector2i) SetByName(name string, value int32) {
	switch name {
	case "x", "X":
		v.X = value
	case "y", "Y":
		v.Y = value
	default:
		panic("Invalid Vector2i component name: " + name)
	}
}

// Zero sets this vector X and Y components to be zero.
// Returns the pointer to this updated vector.
func (v *Vector2i) Zero() *Vector2i {
	v.X = 0
	v.Y = 0
	return v
}

// Copy copies other vector to this one.
// It is equivalent to: *v = *other.
// Returns the pointer to this updated vector.
func (v *Vector2i) Copy(other *Vector2i) *Vector2i {
	v.X = other.X
	v.Y = other.Y
	return v
}

// Add adds other vector to this one.
// Returns the pointer to this updated vector.
func (v *Vector2i) Add(other *Vector2i) *Vector2i {
	v.X += other.X
	v.Y += other.Y
	return v
}

// AddScalar adds scalar s to each component of this vector.
// Returns the pointer to this updated vector.
func (v *Vector2i) AddScalar(s int32) *Vector2i {
	v.X += s
	v.Y += s
	return v
}

// AddVectors adds vectors a and b to this one.
// Returns the pointer to this updated vector.
func (v *Vector2i) AddVectors(a, b *Vector2i) *Vector2i {
	v.X = a.X + b.X
	v.Y = a.Y + b.Y
	return v
}

// Sub subtracts other vector from this one.
// Returns the pointer to this updated vector.
func (v *Vector2i) Sub(other *Vector2i) *Vector2i {
	v.X -= other.X
	v.Y -= other.Y
	return v
}

// SubScalar subtracts scalar s from each component of this vector.
// Returns the pointer to this updated vector.
func (v *Vector2i) SubScalar(s int32) *Vector2i {
	v.X -= s
	v.Y -= s
	return v
}

// SubVectors sets this vector to a - b.
// Returns the pointer to this updated vector.
func (v *Vector2i) SubVectors(a, b *Vector2i) *Vector2i {
	v.X = a.X - b.X
	v.Y = a.Y - b.Y
	return v
}

// Multiply multiplies each component of this vector by the corresponding one from other vector.
// Returns the pointer to this updated vector.
func (v *Vector2i) Multiply(other *Vector2i) *Vector2i {
	v.X *= other.X
	v.Y *= other.Y
	return v
}

// MultiplyScalar multiplies each component of this vector by the scalar s.
// Returns the pointer to this updated vector.
func (v *Vector2i) MultiplyScalar(s int32) *Vector2i {
	v.X *= s
	v.Y *= s
	return v
}

// Divide divides each component of this vector by the corresponding one from other vector.
// Returns the pointer to this updated vector
func (v *Vector2i) Divide(other *Vector2i) *Vector2i {
	v.X /= other.X
	v.Y /= other.Y
	return v
}

// DivideScalar divides each component of this vector by the scalar s.
// If scalar is zero, sets this vector to zero.
// Returns the pointer to this updated vector.
func (v *Vector2i) DivideScalar(scalar int32) *Vector2i {
	if scalar != 0 {
		invScalar := 1 / scalar
		v.X *= invScalar
		v.Y *= invScalar
	} else {
		v.X = 0
		v.Y = 0
	}
	return v
}

// Min sets this vector components to the minimum values of itself and other vector.
// Returns the pointer to this updated vector.
func (v *Vector2i) Min(other *Vector2i) *Vector2i {
	if v.X > other.X {
		v.X = other.X
	}
	if v.Y > other.Y {
		v.Y = other.Y
	}
	return v
}

// Max sets this vector components to the maximum value of itself and other vector.
// Returns the pointer to this updated vector.
func (v *Vector2i) Max(other *Vector2i) *Vector2i {
	if v.X < other.X {
		v.X = other.X
	}
	if v.Y < other.Y {
		v.Y = other.Y
	}
	return v
}

// Clamp sets this vector components to be no less than the corresponding components of min
// and not greater than the corresponding components of max.
// Assumes min < max, if this assumption isn't true it will not operate correctly.
// Returns the pointer to this updated vector.
func (v *Vector2i) Clamp(min, max *Vector2i) *Vector2i {
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
	return v
}

// ClampScalar sets this vector components to be no less than minVal and not greater than maxVal.
// Returns the pointer to this updated vector.
func (v *Vector2i) ClampScalar(minVal, maxVal int32) *Vector2i {
	if v.X < minVal {
		v.X = minVal
	} else if v.X > maxVal {
		v.X = maxVal
	}

	if v.Y < minVal {
		v.Y = minVal
	} else if v.Y > maxVal {
		v.Y = maxVal
	}
	return v
}

// Negate negates each of this vector's components.
// Returns the pointer to this updated vector.
func (v *Vector2i) Negate() *Vector2i {
	v.X = -v.X
	v.Y = -v.Y
	return v
}

// Equals returns if this vector is equal to other.
func (v *Vector2i) Equals(other *Vector2i) bool {
	return (other.X == v.X) && (other.Y == v.Y)
}

// FromArray sets this vector's components from the specified array and offset
// Returns the pointer to this updated vector.
func (v *Vector2i) FromArray(array []int32, offset int) *Vector2i {
	v.X = array[offset]
	v.Y = array[offset+1]
	return v
}

// ToArray copies this vector's components to array starting at offset.
// Returns the array.
func (v *Vector2i) ToArray(array []int32, offset int) []int32 {
	array[offset] = v.X
	array[offset+1] = v.Y
	return array
}
