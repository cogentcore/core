// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

var (
	// UpdateTrace reports a trace of updates that trigger re-rendering.
	// Can be set in PrefsDebug from prefs gui
	UpdateTrace bool

	// RenderTrace reports a trace of the nodes rendering
	// (just printfs to stdout).
	// Can be set in PrefsDebug from prefs gui
	RenderTrace bool

	// LayoutTrace reports a trace of all layouts (just
	// printfs to stdout)
	// Can be set in PrefsDebug from prefs gui
	LayoutTrace bool

	// LayoutTraceDetail provides more detailed info for Layout
	// about the underlying layout computations
	LayoutTraceDetail bool

	// WinEventTrace reports a trace of window events to stdout
	// can be set in PrefsDebug from prefs gui
	// excludes mouse move events
	WinEventTrace = false

	// WinRenderTrace reports the stack trace leading up to win publish events
	// which are expensive -- wrap multiple updates in UpdateStart / End
	// to prevent
	// can be set in PrefsDebug from prefs gui
	WinRenderTrace = false

	// KeyEventTrace reports a trace of keyboard events to stdout
	// can be set in PrefsDebug from prefs gui
	KeyEventTrace = false

	// EventTrace reports a trace of event handing to stdout.
	// can be set in PrefsDebug from prefs gui
	EventTrace = false

	// FocusTrace reports a trace of focus.
	// can be set in PrefsDebug from prefs gui
	FocusTrace = false
)
