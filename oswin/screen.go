// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package oswin

import (
	"image"
	"math"

	"github.com/rcoreilly/goki/ki/kit"
)

// note: fields obtained from QScreen in Qt

// LogicalDPIScale is a scaling factor that can be set by preferences to
// rescale the logical DPI relative to the actual physical DPI, thereby
// scaling the overall density of the display (e.g., smaller numbers produce
// smaller, higher-density displays)
var LogicalDPIScale = float32(0.75)

// Screen contains data about each physical and / or logical screen
type Screen struct {
	// ScreenNumber is the index of this screen in the list of screens
	// maintained under Screen
	ScreenNumber int

	// Geometry contains the geometry of the screen in raw pixels -- all physical screens start at 0,0
	Geometry image.Rectangle

	// Color depth of the screen, in bits
	Depth int

	// LogicalDPI is the logical dots per inch of the window, which is used for all
	// rendering -- subject to zooming effects etc -- see the gi/units package
	// for translating into various other units
	LogicalDPI float32

	// PhysicalDPI is the physical dots per inch of the window, for generating
	// true-to-physical-size output, for example -- see the gi/units package for
	// translating into various other units
	PhysicalDPI float32

	// PhysicalSize is the actual physical size of the screen, in mm
	PhysicalSize image.Point

	// DevicePixelRatio is a multiplier factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	//
	// On OS-X  it is backingScaleFactor, which is 2.0 on "retina" displays
	DevicePixelRatio float32

	RefreshRate float32

	AvailableGeometry        image.Rectangle
	VirtualGeometry          image.Rectangle
	AvailableVirtualGeometry image.Rectangle

	Orientation        ScreenOrientation
	NativeOrientation  ScreenOrientation
	PrimaryOrientation ScreenOrientation

	Name         string
	Manufacturer string
	Model        string
	SerialNumber string
}

// ScreenOrientation is the orientation of the device screen.
type ScreenOrientation int32

const (
	// OrientationUnknown means device orientation cannot be determined.
	//
	// Equivalent on Android to Configuration.ORIENTATION_UNKNOWN
	// and on iOS to:
	//	UIDeviceOrientationUnknown
	//	UIDeviceOrientationFaceUp
	//	UIDeviceOrientationFaceDown
	OrientationUnknown ScreenOrientation = iota

	// Portrait is a device oriented so it is tall and thin.
	//
	// Equivalent on Android to Configuration.ORIENTATION_PORTRAIT
	// and on iOS to:
	//	UIDeviceOrientationPortrait
	//	UIDeviceOrientationPortraitUpsideDown
	Portrait

	// Landscape is a device oriented so it is short and wide.
	//
	// Equivalent on Android to Configuration.ORIENTATION_LANDSCAPE
	// and on iOS to:
	//	UIDeviceOrientationLandscapeLeft
	//	UIDeviceOrientationLandscapeRight
	Landscape

	ScreenOrientationN
)

//go:generate stringer -type=ScreenOrientation

var KiT_ScreenOrientation = kit.Enums.AddEnum(ScreenOrientationN, false, nil)

// LogicalFmPhysicalDPI computes the logical DPI used in actual screen scaling
// based on the LogicalDPIScale factor, and also makes it a multiple of 6 to
// make normal font sizes look best
func LogicalFmPhysicalDPI(pdpi float32) float32 {
	idpi := int(math.Round(float64(pdpi * LogicalDPIScale)))
	mdpi := idpi / 6
	mdpi *= 6
	return float32(mdpi)
}
