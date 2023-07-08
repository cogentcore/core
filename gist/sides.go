// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gist

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/goki/gi/units"
	"github.com/goki/mat32"
)

// Sides contains values for each side of a box
type Sides[T any] struct {
	Top    T `xml:"top" desc:"top value"`
	Right  T `xml:"right" desc:"right value"`
	Bottom T `xml:"bottom" desc:"bottom value"`
	Left   T `xml:"left" desc:"left value"`
}

// NewSides is a helper that creates new sides of the given type
// and calls Set on them.
func NewSides[T any](vals ...T) (Sides[T], error) {
	sides := Sides[T]{}
	err := sides.Set(vals...)
	return sides, err
}

// Set sets the values of the sides from the given list of 0 to 4 values.
// If 0 values are provided, all sides are set to the zero value of the type.
// If 1 value is provided, all sides are set to that value.
// If 2 values are provided, the top and bottom are set to the first value
// and the right and left are set to the second value.
// If 3 values are provided, the top is set to the first value,
// the right and left are set to the second value,
// and the bottom is set to the third value.
// If 4 values are provided, the top is set to the first value,
// the right is set to the second value, the bottom is set
// to the third value, and the left is set to the fourth value.
// If more than 4 values are provided, the behavior is the same
// as with 4 values, but Set also prints and returns
// an error. This error is not critical and does not need to be
// handled, as the values are still set, but it can be if wished.
// This behavior is based on the CSS multi-side setting syntax,
// like that with padding (see https://www.w3schools.com/css/css_padding.asp)
func (s *Sides[T]) Set(vals ...T) error {
	switch len(vals) {
	case 0:
		var zval T
		s.SetAll(zval)
	case 1:
		s.SetAll(vals[0])
	case 2:
		s.SetVert(vals[0])
		s.SetHoriz(vals[1])
	case 3:
		s.Top = vals[0]
		s.SetHoriz(vals[1])
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
		err := fmt.Errorf("sides.Set: expected 0 to 4 values, but got %d", len(vals))
		log.Println(err)
		return err
	}
	return nil
}

// SetVert sets the values for the sides in the vertical direction
// (top and bottom) to the given value
func (s *Sides[T]) SetVert(val T) {
	s.Top = val
	s.Bottom = val
}

// SetHoriz sets the values for the sides in the horizontal direction
// (right and left) to the given value
func (s *Sides[T]) SetHoriz(val T) {
	s.Right = val
	s.Left = val
}

// SetAll sets the values for all of the sides to the given value
func (s *Sides[T]) SetAll(val T) {
	s.Top = val
	s.Right = val
	s.Bottom = val
	s.Left = val
}

// SetStringer is a type that can be set from a string
type SetStringer interface {
	SetString(str string)
}

// SetAny sets the sides from the given value of any type
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

// SetString sets the sides from the given string value
func (s *Sides[T]) SetString(str string) error {
	fields := strings.Fields(str)
	vals := make([]T, len(fields))
	for i, field := range fields {
		ss, ok := any(&vals[i]).(SetStringer)
		if !ok {
			err := errors.New("sides.SetAny: to set from a string, the sides type must implement SetStringer (needs SetString(str string) function)")
			log.Println(err)
			return err
		}
		ss.SetString(field)
	}
	return s.Set(vals...)
}

// SideValues contains units.Value values for each side of a box
type SideValues struct {
	Sides[units.Value]
}

// ToDots converts the values for each of the sides to raw display pixels (dots)
// and sets the Dots field for each of the values. It returns the dot values as a SideFloats.
func (sv *SideValues) ToDots(uc *units.Context) SideFloats {
	return SideFloats{
		Sides: Sides[float32]{
			Top:    sv.Top.ToDots(uc),
			Right:  sv.Right.ToDots(uc),
			Bottom: sv.Bottom.ToDots(uc),
			Left:   sv.Left.ToDots(uc),
		},
	}
}

// Dots returns the dot values of the sides as a SideFloats.
// It does not compute them; see ToDots for that.
func (sv *SideValues) Dots() SideFloats {
	return SideFloats{
		Sides: Sides[float32]{
			Top:    sv.Top.Dots,
			Right:  sv.Right.Dots,
			Bottom: sv.Bottom.Dots,
			Left:   sv.Left.Dots,
		},
	}
}

// ApplyToGeom expands position and size to accommodate the additional space
// in SideValues (e.g., for Padding, Margin)
func (sv *SideValues) ApplyToGeom(pos, sz *mat32.Vec2) {
	sv.ApplyToPos(pos)
	sv.ApplyToSize(sz)
	// pos.Y -= sv.Top.Dots
	// pos.X -= sv.Left.Dots
	// sz.X += sv.Left.Dots + sv.Right.Dots
	// sz.Y += sv.Top.Dots + sv.Bottom.Dots
}

// ApplyToPos adds to the given position the offset in dots caused by the side values
func (sv *SideValues) ApplyToPos(pos *mat32.Vec2) {
	pos.X += sv.Left.Dots
	pos.Y += sv.Top.Dots
}

// ApplyToSize subtracts from the given size the offest in dots caused by the side values
func (sv *SideValues) ApplyToSize(sz *mat32.Vec2) {
	sz.X -= sv.Right.Dots
	sz.Y -= sv.Bottom.Dots
}

// SideFloats contains float32 values for each side of a box
type SideFloats struct {
	Sides[float32]
}

// // ApplyToGeom adds to the given position and subtracts from the given size
// // the offset caused by the side spacing values and returns the resulting
// // position and size.
// func (sf *SideFloats) ApplyToGeom(pos, sz mat32.Vec2) {
// 	sf.ApplyToPos(pos)
// 	sf.ApplyToSize(sz)
// }

// // ApplyToPos adds to the given position the offset caused by
// // the side spacing values and returns the resulting position.
// func (sf *SideFloats) ApplyToPos(pos mat32.Vec2) {
// 	pos.SetAdd(sf.Pos())
// }

// // ApplyToSize subtracts from the given available size the offset
// // caused by the side spacing values and returns the resulting size.
// func (sf *SideFloats) ApplyToSize(sz mat32.Vec2) {
// 	sz.SetSub(sf.Size())
// }

// Pos returns the position offset casued by the side values (Left, Top)
func (sf SideFloats) Pos() mat32.Vec2 {
	return mat32.NewVec2(sf.Left, sf.Top)
}

// Size returns the toal size the side values take up (Left + Right, Top + Bottom)
func (sf SideFloats) Size() mat32.Vec2 {
	return mat32.NewVec2(sf.Left+sf.Right, sf.Top+sf.Bottom)
}
