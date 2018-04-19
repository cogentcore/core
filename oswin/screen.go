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

	"github.com/rcoreilly/goki/ki/kit"
)

// OSScreen is the current oswin Screen for app -- only ever one in effect
var OSScreen Screen

// Screen represents the overall screen hardware, and creates Images, Textures
// and Windows, appropriate for that hardware / OS, and maintains data about
// the physical screen(s)
type Screen interface {
	// NScreens returns the number of different logical and/or physical screens managed under this overall screen hardware
	NScreens() int

	// ScreenData returns screen data for given screen number, or nil if not a
	// valid screen number
	ScreenData(scrN int) *ScreenData

	// NWindows returns the number of windows open for this app
	NWindows() int

	// Window returns given window in list of windows opened under this screen
	Window(win int) Window

	// NewWindow returns a new Window for this screen.
	//
	// A nil opts is valid and means to use the default option values.
	NewWindow(opts *NewWindowOptions) (Window, error)

	// NewImage returns a new Image for this screen.  Images can be drawn upon directly using image and other packages, and have an accessable []byte slice holding the image data
	NewImage(size image.Point) (Image, error)

	// NewTexture returns a new Texture for this screen.  Textures are opaque and could be non-local, but very fast for rendering to windows -- for holding static content
	NewTexture(size image.Point) (Texture, error)
}

// note: fields obtained from QScreen in Qt

// ScreenData contains data about each physical and / or logical screen
type ScreenData struct {
	// ScreenNumber is the index of this screen in the list of screens
	// maintained under Screen
	ScreenNumber int

	// Geometry contains the geometry of the screen -- all physical screens start at 0,0
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
