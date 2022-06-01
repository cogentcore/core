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
	"github.com/goki/vgpu/vdraw"
)

// Window is a double-buffered OS-specific hardware window.
//
// It provides basic GPU support functions, and is currently implemented
// via Vulkan on top of glfw cross-platform window mgmt toolkit (see driver/vkos).
// using the vgpu framework.
//
// The Window maintains its own vdraw.Drawer drawing system for rendering
// bitmap images and filled regions onto the window surface.
//
// The base full-window image should be drawn with a Scale call first,
// followed by any overlay images such as sprites or direct upload
// images already on the GPU (e.g., from 3D render frames)
//
// vgpu.MaxTexturesPerSet = 16 to work cross-platform, meaning that a maximum of 16
// images per descriptor set can be uploaded and be ready to use in one render pass.
// gi.Window uses multiple sets to get around this limitation.
//
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
	// This includes any high DPI factors that may not be used in OS window sizing
	//  (see WinSize for that size).
	Size() image.Point

	// WinSize returns the current size of the window, in OS-specific window manager
	// units that may not include any high DPI factors.
	WinSize() image.Point

	// Position returns the current left-top position of the window relative to
	// underlying screen, in OS-specific window manager coordinates.
	Position() image.Point

	// SetSize sets the size of the window, in OS-specific window manager
	// units that may not include any high DPI factors (DevPixRatio)
	// (i.e., the same units as returned in WinSize)
	SetSize(sz image.Point)

	// SetPixSize sets the size of the window, in actual pixel units.
	// Divides by DevPixRatio before calling SetSize.
	SetPixSize(sz image.Point)

	// SetPos sets the position of the window, in OS window manager
	// coordinates, which may be different from Size() coordinates
	// that reflect high DPI
	SetPos(pos image.Point)

	// SetGeom sets the position and size in one call -- use this if doing
	// both because sequential calls to SetPos and SetSize might fail on some
	// platforms.  Uses OS-specific window manager units that may not include
	// any high DPI factors (DevPixRatio)
	// (i.e., the same units as returned in WinSize, Pos())
	SetGeom(pos image.Point, sz image.Point)

	// Raise requests that the window be at the top of the stack of windows,
	// and receive focus.  If it is iconified, it will be de-iconified.  This
	// is the only supported mechanism for de-iconifying.
	Raise()

	// Minimize requests that the window be iconified, making it no longer
	// visible or active -- rendering should not occur for minimized windows.
	Minimize()

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

	// IsMinimized returns true if this window is minimized.  See also IsVisible()
	IsMinimized() bool

	// IsFocus returns true if this window is focused (will receive keyboard input etc).
	IsFocus() bool

	// IsClosed returns true if this window has been closed (but some threads
	// may have not received the news yet)
	IsClosed() bool

	// IsVisible returns true if this window is not closed or minimized and
	// there are active screens
	IsVisible() bool

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

	// RunOnWin runs given function on the window's own separate goroutine
	RunOnWin(f func())

	// GoRunOnWin runs given function on the window's unique locked thread
	// and returns immediately.
	GoRunOnWin(f func())

	// SendEmptyEvent sends an empty, blank event to this window, which just has
	// the effect of pushing the system along during cases when the window
	// event loop needs to be "pinged" to get things moving along..
	SendEmptyEvent()

	// Handle returns the driver-specific handle for this window.
	// Currently, for all platforms, this is *glfw.Window, but that
	// cannot always be assumed.  Only provided for unforeseen emergency use --
	// please file an Issue for anything that should be added to Window
	// interface.
	Handle() interface{}

	// OSHandle returns the OS-specific underlying window handle:
	// MacOS: NSWindow*, Windows:  HWND, LinuxX11: X11Window
	OSHandle() uintptr

	// Sets the mouse position to given values
	SetMousePos(x, y float64)

	// Sets the mouse cursor to be enabled (true by default) or disabled (false).
	// If disabled, setting raw to true will enable raw mouse movement
	// which can provide better control in a game environment (not avail on Mac).
	SetCursorEnabled(enabled, raw bool)

	// Drawer returns the drawing system attached to this window surface.
	// This is typically used for high-performance rendering to the surface.
	Drawer() *vdraw.Drawer

	// SetDestroyGPUResourcesFunc sets the given function
	// that will be called on the main thread just prior
	// to destroying the drawer and surface.
	SetDestroyGPUResourcesFunc(f func())

	EventDeque
}

// WindowBase provides a base-level implementation of the generic data aspects
// of the window, including maintaining the current window size and dpi
type WindowBase struct {
	Nm          string
	Titl        string
	Pos         image.Point
	WnSize      image.Point // window-manager coords
	PxSize      image.Point // pixel size
	DevPixRatio float32
	PhysDPI     float32
	LogDPI      float32
	Par         interface{}
	Flag        int64
	// set this to a function that will destroy GPU resources
	// in the main thread prior to destroying the drawer
	// and the surface -- otherwise it is difficult to
	// ensure that the proper ordering of destruction applies.
	DestroyGPUfunc func()
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

func (w *WindowBase) SetDestroyGPUResourcesFunc(f func()) {
	w.DestroyGPUfunc = f
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

var KiT_WindowFlags = kit.Enums.AddEnum(WindowFlagsN, kit.BitFlag, nil)

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

	// Note: StdPixels IS the default behavior at this point
	// so we don't need to do any correction factors to deal
	// with it!

	if o.Size.X > scsz.X {
		o.Size.X = scsz.X
	}
	if o.Size.Y > scsz.Y {
		o.Size.Y = scsz.Y
	}

	// these are windows-specific special numbers for minimized windows
	// can be sent here for WinGeom saved geom.
	if o.Pos.X == -32000 {
		o.Pos.X = 0
	}
	if o.Pos.Y == -32000 {
		o.Pos.Y = 50
	}

	if o.Pos.X == 0 && o.Pos.Y == 0 {
		nw := TheApp.NWindows()
		if nw > 0 {
			lastw := TheApp.Window(nw - 1)
			lsz := lastw.WinSize()
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
