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
	"unicode/utf8"

	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// Window is a double-buffered OS-specific hardware window.  Windows are
// generally always visible unless specifically iconified (i.e., there is no
// explicit mechanism to show the window).
type Window interface {

	// Name returns the name of the window -- name is used strictly for
	// internal tracking and finding of windows -- see Title for the displayed
	// title of the window.
	Name() string

	// SetName sets the name of the window.
	SetName(name string)

	// Title returns the current title of the window, which is displayed in the GUI.
	Title() string

	// SetTitle sets the current title of the window, which is displayed in the GUI.
	SetTitle(title string)

	// Size returns the current size of the window, in raw underlying dots / pixels.
	Size() image.Point

	// SetSize sets the size of the window, in raw underlying dots / pixels.
	SetSize(sz image.Point)

	// SetPos sets the position of the window, in raw underlying dots /
	// pixels.
	SetPos(pos image.Point)

	// SetGeom sets the position and size in one call -- use this if doing
	// both because sequential calls to SetPos and SetSize might fail on some
	// platforms
	SetGeom(pos image.Point, sz image.Point)

	// Raise requests that the window be at the top of the stack of windows,
	// and receive focus.  If it is iconified, it will be de-iconified.  This
	// is the only supported mechanism for de-iconifying.
	Raise()

	// Minimize requests that the window be iconified, making it no longer
	// visible or active -- rendering should not occur for minimized windows.
	Minimize()

	// Position returns the current lef-top position of the window relative to
	// underlying screen, in raw underlying dots / pixels.
	Position() image.Point

	// PhysicalDPI is the physical dots per inch of the window, for generating
	// true-to-physical-size output, for example -- see the gi/units package for
	// translating into various other units.
	PhysicalDPI() float32

	// LogicalDPI returns the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units --
	// physical DPI can be found in the Screen.
	LogicalDPI() float32

	// SetLogicalDPI sets the current logical dots-per-inch resolution of the
	// window, which should be used for most conversion of standard units --
	// physical DPI can be found in the Screen.
	SetLogicalDPI(dpi float32)

	// Screen returns the screen for this window, with all the relevant
	// information about its properties.
	Screen() *Screen

	// Parent returns the parent object of a given window -- for GoGi it is a
	// gi.Window but could be something else in other frameworks.
	Parent() interface{}

	// SetParent sets the parent object of a given window -- for GoGi it is a
	// gi.Window but could be something else in other frameworks.
	SetParent(par interface{})

	// MainMenu returns the OS-level main menu for this window -- this is
	// currently for MacOS only -- returns nil for others.
	MainMenu() MainMenu

	// Flags returns the bit flags for this window's properties set according
	// to WindowFlags bits.
	Flags() int64

	// IsDialog returns true if this is a dialog window.
	IsDialog() bool

	// IsModal returns true if this is a modal window (blocks input to other windows).
	IsModal() bool

	// IsTool returns true if this is a tool window.
	IsTool() bool

	// IsFullscreen returns true if this is a fullscreen window.
	IsFullscreen() bool

	// IsMinimized returns true if this window is minimized.
	IsMinimized() bool

	// IsFocused returns true if this window is focused (will receive keyboard
	// input etc).
	IsFocus() bool

	// SetCloseReqFunc sets the function that is called whenever there is a
	// request to close the window (via a OS or a call to CloseReq() method).  That
	// function can then adjudicate whether and when to actually call Close.
	SetCloseReqFunc(fun func(win Window))

	// SetCloseCleanFunc sets the function that is called whenever window is
	// actually about to close (irrevocably) -- can do any necessary
	// last-minute cleanup here.
	SetCloseCleanFunc(fun func(win Window))

	// CloseReq is a close request, triggered either by OS or user call (e.g.,
	// via Close menu action) -- calls function previously-registered by
	// SetCloseReqFunc, which is then solely responsible for actually calling
	// Close.
	CloseReq()

	// CloseClean calls the function setup in SetCloseCleanFunc and does other
	// window cleanup -- called on way to closing.
	CloseClean()

	// Close requests that the window be closed. The behavior of the Window
	// after this, whether calling its methods or passing it as an argument,
	// is undefined.  See App.Quit methods to quit overall app.
	Close()

	EventDeque

	Uploader

	Drawer

	// Publish flushes any pending Upload and Draw calls to the window, and
	// swaps the back buffer to the front.
	Publish() PublishResult
}

// PublishResult is the result of an Window.Publish call.
type PublishResult struct {
	// BackImagePreserved is whether the contents of the back buffer was
	// preserved. If false, the contents are undefined.
	BackImagePreserved bool
}

// WindowBase provides a base-level implementation of the generic data aspects
// of the window, including maintaining the current window size and dpi
type WindowBase struct {
	Nm      string
	Titl    string
	Sz      image.Point
	Pos     image.Point
	PhysDPI float32
	LogDPI  float32
	Scrn    *Screen
	Par     interface{}
	Flag    int64
}

func (w WindowBase) Name() string {
	return w.Nm
}

func (w *WindowBase) SetName(name string) {
	w.Nm = name
}

func (w WindowBase) Title() string {
	return w.Titl
}

func (w WindowBase) Parent() interface{} {
	return w.Par
}

func (w *WindowBase) SetParent(parent interface{}) {
	w.Par = parent
}

func (w *WindowBase) Flags() int64 {
	return w.Flag
}

func (w *WindowBase) IsDialog() bool {
	return bitflag.HasAtomic(&w.Flag, int(Dialog))
}

func (w *WindowBase) IsModal() bool {
	return bitflag.HasAtomic(&w.Flag, int(Modal))
}

func (w *WindowBase) IsTool() bool {
	return bitflag.HasAtomic(&w.Flag, int(Tool))
}

func (w *WindowBase) IsFullscreen() bool {
	return bitflag.HasAtomic(&w.Flag, int(Fullscreen))
}

func (w *WindowBase) IsMinimized() bool {
	return bitflag.HasAtomic(&w.Flag, int(Minimized))
}

func (w *WindowBase) IsFocus() bool {
	return bitflag.HasAtomic(&w.Flag, int(Focus))
}

////////////////////////////////////////////////////////////////////////////
// WindowOptions

// Qt options: http://doc.qt.io/qt-5/qt.html#WindowType-enum

// WindowFlags contains all the binary properties of a window -- by default
// with no other relevant flags a window is a main top-level window.
type WindowFlags int32

const (
	// Dialog indicates that this is a temporary, pop-up window.
	Dialog WindowFlags = iota

	// Modal indicates that this dialog window blocks events going to other
	// windows until it is closed.
	Modal

	// Tool indicates that this is a floating tool window that has minimized
	// window decoration.
	Tool

	// Fullscreen indicates a window that occupies the entire screen.
	Fullscreen

	// Minimized indicates a window reduced to an icon, or otherwise no longer
	// visible or active.  Otherwise, the window should be assumed to be
	// visible.
	Minimized

	// Focus indicates that the window has the focus.
	Focus

	WindowFlagsN
)

//go:generate stringer -type=WindowFlags

var KiT_WindowFlags = kit.Enums.AddEnum(WindowFlagsN, true, nil) // bitflags

// NewWindowOptions are optional arguments to NewWindow.
type NewWindowOptions struct {
	// Size specifies the dimensions of the new window, either in raw pixels
	// or std 96 dpi pixels depending on StdPixels. If Width or Height are
	// zero, a driver-dependent default will be used for each zero value
	// dimension
	Size image.Point

	// StdPixels means use standardized "pixel" units for the window size (96
	// per inch), not the actual underlying raw display dot pixels
	StdPixels bool

	// Pos specifies the position of the window, if non-zero -- always in
	// device-specific raw pixels
	Pos image.Point

	// Title specifies the window title.
	Title string

	// Flags can be set using WindowFlags to request different types of windows
	Flags int64
}

func (o *NewWindowOptions) SetDialog() {
	bitflag.Set(&o.Flags, int(Dialog))
}

func (o *NewWindowOptions) SetModal() {
	bitflag.Set(&o.Flags, int(Modal))
}

func (o *NewWindowOptions) SetTool() {
	bitflag.Set(&o.Flags, int(Tool))
}

func (o *NewWindowOptions) SetFullscreen() {
	bitflag.Set(&o.Flags, int(Fullscreen))
}

func WindowFlagsToBool(flags int64) (dialog, modal, tool, fullscreen bool) {
	dialog = bitflag.Has(flags, int(Dialog))
	modal = bitflag.Has(flags, int(Modal))
	tool = bitflag.Has(flags, int(Tool))
	fullscreen = bitflag.Has(flags, int(Fullscreen))
	return
}

// GetTitle returns a sanitized form of o.Title. In particular, its length will
// not exceed 4096, and it may be further truncated so that it is valid UTF-8
// and will not contain the NUL byte.
//
// o may be nil, in which case "" is returned.
func (o *NewWindowOptions) GetTitle() string {
	if o == nil {
		return ""
	}
	return sanitizeUTF8(o.Title, 4096)
}

func sanitizeUTF8(s string, n int) string {
	if n < len(s) {
		s = s[:n]
	}
	i := 0
	for i < len(s) {
		r, n := utf8.DecodeRuneInString(s[i:])
		if r == 0 || (r == utf8.RuneError && n == 1) {
			break
		}
		i += n
	}
	return s[:i]
}

// Fixup fills in defaults and updates everything based on current screen and
// window context Specific hardware can fine-tune this as well, in driver code
func (o *NewWindowOptions) Fixup() {
	sc := TheApp.Screen(0)
	scsz := sc.Geometry.Size()

	dialog, modal, _, _ := WindowFlagsToBool(o.Flags)

	if o.Size.X <= 0 {
		o.StdPixels = false
		o.Size.X = int(0.8 * float32(scsz.X))
	}
	if o.Size.Y <= 0 {
		o.StdPixels = false
		o.Size.Y = int(0.8 * float32(scsz.Y))
	}

	if o.StdPixels {
		winDPI := sc.LogicalDPI
		o.Size.X = int(float32(o.Size.X) * (winDPI / 96.0))
		o.Size.Y = int(float32(o.Size.Y) * (winDPI / 96.0))
	}

	if o.Size.X > scsz.X {
		o.Size.X = scsz.X
	}
	if o.Size.Y > scsz.Y {
		o.Size.Y = scsz.Y
	}

	if o.Pos.X == 0 && o.Pos.Y == 0 {
		nw := TheApp.NWindows()
		if nw > 0 {
			lastw := TheApp.Window(nw - 1)
			lsz := lastw.Size()
			lp := lastw.Position()

			nwbig := o.Size.X > lsz.X || o.Size.Y > lsz.Y

			if modal || dialog || !nwbig { // place centered on top of current
				ctrx := lp.X + (lsz.X / 2)
				ctry := lp.Y + (lsz.Y / 2)
				o.Pos.X = ctrx - o.Size.X/2
				o.Pos.Y = ctry - o.Size.Y/2
			} else { // cascade to right
				o.Pos.X = lp.X + lsz.X // tile to right -- could depend on orientation
				o.Pos.Y = lp.Y + 72    // and move down a bit
			}
		} else { // center in screen
			o.Pos.X = scsz.X/2 - o.Size.X/2
			o.Pos.Y = scsz.Y/2 - o.Size.Y/2
		}
	}

	// final sanity fixes
	if o.Pos.X+o.Size.X > scsz.X {
		o.Pos.X = scsz.X - o.Size.X
	}
	if o.Pos.Y+o.Size.Y > scsz.Y {
		o.Pos.Y = scsz.Y - o.Size.Y
	}
}
