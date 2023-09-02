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

	"goki.dev/ki/v2/kit"
)

var (
	// ZoomFactor is a multiplier on screen LogicalDPI
	ZoomFactor = float32(1.0)

	// LogicalDPIScale is the default scaling factor for Logical DPI
	// as a multiplier on Physical DPI.
	// Smaller numbers produce smaller font sizes etc.
	LogicalDPIScale = float32(1.0)

	// LogicalDPIScales are per-screen name versions of LogicalDPIScale
	// these can be set from preferences (as in gi/prefs) on a per-screen
	// basis.
	LogicalDPIScales map[string]float32
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
	// which is used for all rendering.  It is set as a function of the
	// global ZoomFactor and the LogicalDPIScale, either the global
	// or per-screen name version if it exists.
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

var TypeScreenOrientation = kit.Enums.AddEnum(ScreenOrientationN, kit.NotBitFlag, nil)

// LogicalFmPhysicalDPI computes the logical DPI used in actual screen scaling
// based on the given logical DPI scale factor (logScale), and also makes it a
// multiple of 6 to make normal font sizes look best.
func LogicalFmPhysicalDPI(logScale, pdpi float32) float32 {
	idpi := int(math.Round(float64(pdpi * logScale)))
	mdpi := idpi / 6
	mdpi *= 6
	return float32(mdpi)
}

// SetLogicalDPIScale sets the LogicalDPIScale factor for given screen name
func SetLogicalDPIScale(scrnName string, dpiScale float32) {
	if LogicalDPIScales == nil {
		LogicalDPIScales = make(map[string]float32)
	}
	LogicalDPIScales[scrnName] = dpiScale
}

// UpdateLogicalDPI updates the logical DPI of the screen
// based on ZoomFactor and LogicalDPIScale (per screen if exists)
func (sc *Screen) UpdateLogicalDPI() {
	dpisc := LogicalDPIScale
	if LogicalDPIScales != nil {
		if dsc, has := LogicalDPIScales[sc.Name]; has {
			dpisc = dsc
		}
	}
	sc.LogicalDPI = LogicalFmPhysicalDPI(ZoomFactor*dpisc, sc.PhysicalDPI)
}

// WinSizeToPix returns window manager size units
// (where DevicePixelRatio is ignored) converted to pixel units --
// i.e., multiply by DevicePixelRatio
func (sc *Screen) WinSizeToPix(sz image.Point) image.Point {
	var psz image.Point
	psz.X = int(float32(sz.X) * sc.DevicePixelRatio)
	psz.Y = int(float32(sz.Y) * sc.DevicePixelRatio)
	return psz
}

// WinSizeFmPix returns window manager size units
// (where DevicePixelRatio is ignored) converted from pixel units --
// i.e., divide by DevicePixelRatio
func (sc *Screen) WinSizeFmPix(sz image.Point) image.Point {
	var wsz image.Point
	wsz.X = int(float32(sz.X) / sc.DevicePixelRatio)
	wsz.Y = int(float32(sz.Y) / sc.DevicePixelRatio)
	return wsz
}

// ConstrainWinGeom constrains window geometry to fit in the screen.
// Size is in pixel units.
func (sc *Screen) ConstrainWinGeom(sz, pos image.Point) (csz, cpos image.Point) {
	scsz := sc.Geometry.Size() // in window coords size

	// options size are in pixel sizes, logic below works in window sizes
	csz = sc.WinSizeFmPix(sz)
	cpos = pos

	// fmt.Printf("sz: %v  csz: %v  scsz: %v\n", sz, csz, scsz)
	// fmt.Println(string(debug.Stack()))
	if csz.X > scsz.X {
		csz.X = scsz.X - 10
		// fmt.Println("constrain x:", csz.X)
	}
	if csz.Y > scsz.Y {
		csz.Y = scsz.Y - 10
		// fmt.Println("constrain y:", csz.Y)
	}

	// these are windows-specific special numbers for minimized windows
	// can be sent here for WinGeom saved geom.
	if cpos.X == -32000 {
		cpos.X = 0
	}
	if cpos.Y == -32000 {
		cpos.Y = 50
	}

	// don't go off the edge
	if cpos.X+csz.X > scsz.X {
		cpos.X = scsz.X - csz.X
	}
	if cpos.Y+csz.Y > scsz.Y {
		cpos.Y = scsz.Y - csz.Y
	}
	if cpos.X < 0 {
		cpos.X = 0
	}
	if cpos.Y < 0 {
		cpos.Y = 0
	}

	csz = sc.WinSizeToPix(csz)
	return
}

// InitScreenLogicalDPIFunc is a function that can be set to initialize the
// screen LogicalDPI values based on user preferences etc.  Called just before
// first window is opened.
var InitScreenLogicalDPIFunc func()
