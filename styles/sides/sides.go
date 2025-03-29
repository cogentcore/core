// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sides provides flexible representation of box sides
// or corners, with either a single value for all, or different values
// for subsets.
package sides

//go:generate core generate

import (
	"fmt"
	"image/color"
	"strings"

	"log/slog"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles/units"
)

// Indexes provides names for the Sides in order defined
type Indexes int32 //enums:enum

const (
	Top Indexes = iota
	Right
	Bottom
	Left
)

// Sides contains values for each side or corner of a box.
// If Sides contains sides, the struct field names correspond
// directly to the side values (ie: Top = top side value).
// If Sides contains corners, the struct field names correspond
// to the corners as follows: Top = top left, Right = top right,
// Bottom = bottom right, Left = bottom left.
type Sides[T any] struct { //types:add

	// top/top-left value
	Top T

	// right/top-right value
	Right T

	// bottom/bottom-right value
	Bottom T

	// left/bottom-left value
	Left T
}

// NewSides is a helper that creates new sides/corners of the given type
// and calls Set on them with the given values.
func NewSides[T any](vals ...T) *Sides[T] {
	return (&Sides[T]{}).Set(vals...)
}

// Set sets the values of the sides/corners from the given list of 0 to 4 values.
// If 0 values are provided, all sides/corners are set to the zero value of the type.
// If 1 value is provided, all sides/corners are set to that value.
// If 2 values are provided, the top/top-left and bottom/bottom-right are set to the first value
// and the right/top-right and left/bottom-left are set to the second value.
// If 3 values are provided, the top/top-left is set to the first value,
// the right/top-right and left/bottom-left are set to the second value,
// and the bottom/bottom-right is set to the third value.
// If 4 values are provided, the top/top-left is set to the first value,
// the right/top-right is set to the second value, the bottom/bottom-right is set
// to the third value, and the left/bottom-left is set to the fourth value.
// If more than 4 values are provided, the behavior is the same
// as with 4 values, but Set also logs a programmer error.
// This behavior is based on the CSS multi-side/corner setting syntax,
// like that with padding and border-radius (see https://www.w3schools.com/css/css_padding.asp
// and https://www.w3schools.com/cssref/css3_pr_border-radius.php)
func (s *Sides[T]) Set(vals ...T) *Sides[T] {
	switch len(vals) {
	case 0:
		var zval T
		s.SetAll(zval)
	case 1:
		s.SetAll(vals[0])
	case 2:
		s.SetVertical(vals[0])
		s.SetHorizontal(vals[1])
	case 3:
		s.Top = vals[0]
		s.SetHorizontal(vals[1])
		s.Bottom = vals[2]
	case 4:
		s.Top = vals[0]
		s.Right = vals[1]
		s.Bottom = vals[2]
		s.Left = vals[3]
	default:
		s.Top = vals[0]
		s.Right = vals[1]
		s.Bottom = vals[2]
		s.Left = vals[3]
		slog.Error("programmer error: sides.Set: expected 0 to 4 values, but got", "numValues", len(vals))
	}
	return s
}

// Zero sets the values of all of the sides to zero.
func (s *Sides[T]) Zero() *Sides[T] {
	s.Set()
	return s
}

// SetVertical sets the values for the sides/corners in the
// vertical/diagonally descending direction
// (top/top-left and bottom/bottom-right) to the given value
func (s *Sides[T]) SetVertical(val T) *Sides[T] {
	s.Top = val
	s.Bottom = val
	return s
}

// SetHorizontal sets the values for the sides/corners in the
// horizontal/diagonally ascending direction
// (right/top-right and left/bottom-left) to the given value
func (s *Sides[T]) SetHorizontal(val T) *Sides[T] {
	s.Right = val
	s.Left = val
	return s
}

// SetAll sets the values for all of the sides/corners
// to the given value
func (s *Sides[T]) SetAll(val T) *Sides[T] {
	s.Top = val
	s.Right = val
	s.Bottom = val
	s.Left = val
	return s
}

// SetTop sets the top side to the given value
func (s *Sides[T]) SetTop(top T) *Sides[T] {
	s.Top = top
	return s
}

// SetRight sets the right side to the given value
func (s *Sides[T]) SetRight(right T) *Sides[T] {
	s.Right = right
	return s
}

// SetBottom sets the bottom side to the given value
func (s *Sides[T]) SetBottom(bottom T) *Sides[T] {
	s.Bottom = bottom
	return s
}

// SetLeft sets the left side to the given value
func (s *Sides[T]) SetLeft(left T) *Sides[T] {
	s.Left = left
	return s
}

// SetAny sets the sides/corners from the given value of any type
func (s *Sides[T]) SetAny(a any) error {
	switch val := a.(type) {
	case Sides[T]:
		*s = val
	case *Sides[T]:
		*s = *val
	case T:
		s.SetAll(val)
	case *T:
		s.SetAll(*val)
	case []T:
		s.Set(val...)
	case *[]T:
		s.Set(*val...)
	case string:
		return s.SetString(val)
	default:
		return s.SetString(fmt.Sprint(val))
	}
	return nil
}

// SetString sets the sides/corners from the given string value
func (s *Sides[T]) SetString(str string) error {
	fields := strings.Fields(str)
	vals := make([]T, len(fields))
	for i, field := range fields {
		ss, ok := any(&vals[i]).(reflectx.SetStringer)
		if !ok {
			err := fmt.Errorf("(Sides).SetString('%s'): to set from a string, the sides type (%T) must implement reflectx.SetStringer (needs SetString(str string) error function)", str, s)
			slog.Error(err.Error())
			return err
		}
		err := ss.SetString(field)
		if err != nil {
			nerr := fmt.Errorf("(Sides).SetString('%s'): error setting sides of type %T from string: %w", str, s, err)
			slog.Error(nerr.Error())
			return nerr
		}
	}
	s.Set(vals...)
	return nil
}

// AreSame returns whether all of the sides/corners are the same
func AreSame[T comparable](s Sides[T]) bool {
	return s.Right == s.Top && s.Bottom == s.Top && s.Left == s.Top
}

// AreZero returns whether all of the sides/corners are equal to zero
func AreZero[T comparable](s Sides[T]) bool {
	var zv T
	return s.Top == zv && s.Right == zv && s.Bottom == zv && s.Left == zv
}

// Values contains units.Value values for each side/corner of a box
type Values struct { //types:add
	Sides[units.Value]
}

// NewValues is a helper that creates new side/corner values
// and calls Set on them with the given values.
func NewValues(vals ...units.Value) Values {
	sides := Sides[units.Value]{}
	sides.Set(vals...)
	return Values{sides}
}

// ToDots converts the values for each of the sides/corners
// to raw display pixels (dots) and sets the Dots field for each
// of the values. It returns the dot values as a Floats.
func (sv *Values) ToDots(uc *units.Context) Floats {
	return NewFloats(
		sv.Top.ToDots(uc),
		sv.Right.ToDots(uc),
		sv.Bottom.ToDots(uc),
		sv.Left.ToDots(uc),
	)
}

// Dots returns the dot values of the sides/corners as a Floats.
// It does not compute them; see ToDots for that.
func (sv Values) Dots() Floats {
	return NewFloats(
		sv.Top.Dots,
		sv.Right.Dots,
		sv.Bottom.Dots,
		sv.Left.Dots,
	)
}

// Floats contains float32 values for each side/corner of a box
type Floats struct { //types:add
	Sides[float32]
}

// NewFloats is a helper that creates new side/corner floats
// and calls Set on them with the given values.
func NewFloats(vals ...float32) Floats {
	sides := Sides[float32]{}
	sides.Set(vals...)
	return Floats{sides}
}

// Add adds the side floats to the
// other side floats and returns the result
func (sf Floats) Add(other Floats) Floats {
	return NewFloats(
		sf.Top+other.Top,
		sf.Right+other.Right,
		sf.Bottom+other.Bottom,
		sf.Left+other.Left,
	)
}

// Sub subtracts the other side floats from
// the side floats and returns the result
func (sf Floats) Sub(other Floats) Floats {
	return NewFloats(
		sf.Top-other.Top,
		sf.Right-other.Right,
		sf.Bottom-other.Bottom,
		sf.Left-other.Left,
	)
}

// MulScalar multiplies each side by the given scalar value
// and returns the result.
func (sf Floats) MulScalar(s float32) Floats {
	return NewFloats(
		sf.Top*s,
		sf.Right*s,
		sf.Bottom*s,
		sf.Left*s,
	)
}

// Min returns a new side floats containing the
// minimum values of the two side floats
func (sf Floats) Min(other Floats) Floats {
	return NewFloats(
		math32.Min(sf.Top, other.Top),
		math32.Min(sf.Right, other.Right),
		math32.Min(sf.Bottom, other.Bottom),
		math32.Min(sf.Left, other.Left),
	)
}

// Max returns a new side floats containing the
// maximum values of the two side floats
func (sf Floats) Max(other Floats) Floats {
	return NewFloats(
		math32.Max(sf.Top, other.Top),
		math32.Max(sf.Right, other.Right),
		math32.Max(sf.Bottom, other.Bottom),
		math32.Max(sf.Left, other.Left),
	)
}

// Round returns a new side floats with each side value
// rounded to the nearest whole number.
func (sf Floats) Round() Floats {
	return NewFloats(
		math32.Round(sf.Top),
		math32.Round(sf.Right),
		math32.Round(sf.Bottom),
		math32.Round(sf.Left),
	)
}

// Pos returns the position offset casued by the side/corner values (Left, Top)
func (sf Floats) Pos() math32.Vector2 {
	return math32.Vec2(sf.Left, sf.Top)
}

// Size returns the toal size the side/corner values take up (Left + Right, Top + Bottom)
func (sf Floats) Size() math32.Vector2 {
	return math32.Vec2(sf.Left+sf.Right, sf.Top+sf.Bottom)
}

// ToValues returns the side floats a
// Values composed of [units.UnitDot] values
func (sf Floats) ToValues() Values {
	return NewValues(
		units.Dot(sf.Top),
		units.Dot(sf.Right),
		units.Dot(sf.Bottom),
		units.Dot(sf.Left),
	)
}

// Colors contains color values for each side/corner of a box
type Colors struct { //types:add
	Sides[color.RGBA]
}

// NewColors is a helper that creates new side/corner colors
// and calls Set on them with the given values.
// It does not return any error values and just logs them.
func NewColors(vals ...color.RGBA) Colors {
	sides := Sides[color.RGBA]{}
	sides.Set(vals...)
	return Colors{sides}
}

// SetAny sets the sides/corners from the given value of any type
func (s *Colors) SetAny(a any, base color.Color) error {
	switch val := a.(type) {
	case Sides[color.RGBA]:
		s.Sides = val
	case *Sides[color.RGBA]:
		s.Sides = *val
	case color.RGBA:
		s.SetAll(val)
	case *color.RGBA:
		s.SetAll(*val)
	case []color.RGBA:
		s.Set(val...)
	case *[]color.RGBA:
		s.Set(*val...)
	case string:
		return s.SetString(val, base)
	default:
		return s.SetString(fmt.Sprint(val), base)
	}
	return nil
}

// SetString sets the sides/corners from the given string value
func (s *Colors) SetString(str string, base color.Color) error {
	fields := strings.Fields(str)
	vals := make([]color.RGBA, len(fields))
	for i, field := range fields {
		clr, err := colors.FromString(field, base)
		if err != nil {
			nerr := fmt.Errorf("(Colors).SetString('%s'): error setting sides of type %T from string: %w", str, s, err)
			slog.Error(nerr.Error())
			return nerr
		}
		vals[i] = clr
	}
	s.Set(vals...)
	return nil
}
