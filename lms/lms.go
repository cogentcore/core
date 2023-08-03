// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lms

import "github.com/goki/ki/kit"

// Components are different components of the LMS space
// including opponent contrasts and grey
type Components int

//go:generate stringer -type=Components

var TypeComponents = kit.Enums.AddEnum(ComponentsN, kit.NotBitFlag, nil)

func (ev Components) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *Components) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

const (
	// Long wavelength = Red component
	LC Components = iota

	// Medium wavelength = Green component
	MC

	// Short wavelength = Blue component
	SC

	// Long + Medium wavelength = Yellow component
	LMC

	// L - M opponent contrast: Red vs. Green
	LvMC

	// S - L+M opponent contrast: Blue vs. Yellow
	SvLMC

	// achromatic response (grey scale lightness)
	GREY

	// number of components
	ComponentsN
)
