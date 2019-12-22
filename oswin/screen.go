// Copyright (c) 2018, The GoKi Authors. All rights reserved.
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

	"github.com/goki/ki/kit"
)

// note: fields obtained from QScreen in Qt

// Screen contains data about each physical and / or logical screen
type Screen struct {
	// ScreenNumber is the index of this screen in the list of screens
	// maintained under Screen.
	ScreenNumber int

	// Geometry contains the geometry of the screen in window manager
	// size units, which may not be same as raw pixels
	Geometry image.Rectangle

	// DevicePixelRatio is a factor that scales the screen's
	// "natural" pixel coordinates into actual device pixels.
	// On OS-X  it is backingScaleFactor = 2.0 on "retina"
	DevicePixelRatio float32

	//	PixSize is the number of actual pixels in the screen
	// computed as Size * DevicePixelRatio
	PixSize image.Point

	// PhysicalSize is the actual physical size of the screen, in mm.
	PhysicalSize image.Point

	// LogicalDPI is the logical dots per inch of the screen,
	// which is used for all rendering -- subject to zooming
	// effects etc -- see the gi/units package for translating
	// into various other units.
	LogicalDPI float32

	// PhysicalDPI is the physical dots per inch of the screen,
	// for generating true-to-physical-size output, for example
	// see the gi/units package for translating into various other
	// units.
	PhysicalDPI float32

	// Color depth of the screen, in bits.
	Depth int

	// Refresh rate in Hz
	RefreshRate float32

	// todo: not using these yet
	// AvailableGeometry        image.Rectangle
	// VirtualGeometry          image.Rectangle
	// AvailableVirtualGeometry image.Rectangle

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

var KiT_ScreenOrientation = kit.Enums.AddEnum(ScreenOrientationN, kit.NotBitFlag, nil)

// LogicalFmPhysicalDPI computes the logical DPI used in actual screen scaling
// based on the given logical DPI scale factor (logScale), and also makes it a
// multiple of 6 to make normal font sizes look best.
func LogicalFmPhysicalDPI(logScale, pdpi float32) float32 {
	idpi := int(math.Round(float64(pdpi * logScale)))
	mdpi := idpi / 6
	mdpi *= 6
	return float32(mdpi)
}

// InitScreenLogicalDPIFunc is a function that can be set to initialize the
// screen LogicalDPI values based on user preferences etc.  Called just before
// first window is opened.
var InitScreenLogicalDPIFunc func()
