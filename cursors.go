// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursors provides Go constant names for cursors as SVG files.
package cursors

//go:generate enumgen

import (
	"embed"
	"image"

	"goki.dev/enums"
)

// Cursors contains all of the default embedded svg cursors
//
//go:embed png/*/*.png
var Cursors embed.FS

// Cursor represents a cursor
type Cursor int32 //enums:enum -transform kebab

// Cursor constants, with names based on CSS (see https://developer.mozilla.org/en-US/docs/Web/CSS/cursor)
// TODO: maybe add NoDrop and AllScroll
const (
	// None indicates no preference for a cursor; will typically be inherited
	None Cursor = iota
	// Arrow is a standard arrow cursor, which is the default window cursor
	Arrow
	// ContextMenu indicates that a context menu is available
	ContextMenu
	// Help indicates that help information is available
	Help
	// Pointer is a pointing hand that indicates a link or an interactive element
	Pointer
	// Progress indicates that the app is busy in the background, but can still be
	// interacted with (use [Wait] to indicate that it can't be interacted with)
	Progress
	// Wait indicates that the app is busy and can not be interacted with
	// (use [Progress] to indicate that it can be interacted with)
	Wait
	// Cell indicates a table cell, especially one that can be selected
	Cell
	// Crosshair is a cross cursor that typically indicates precision selection, such as in an image
	Crosshair
	// Text is an I-Beam that indicates text that can be selected
	Text
	// VerticalText is a sideways I-Beam that indicates vertical text that can be selected
	VerticalText
	// Alias indicates that a shortcut or alias will be created
	Alias
	// Copy indicates that a copy of something will be created
	Copy
	// Move indicates that something is being moved
	Move
	// NotAllowed indicates that something can not be done
	NotAllowed
	// Grab indicates that something can be grabbed
	Grab
	// Grabbing indicates that something is actively being grabbed
	Grabbing
	// ResizeCol indicates that something can be resized in the horizontal direction
	ResizeCol
	// ResizeRow indicates that something can be resized in the vertical direction
	ResizeRow
	// ResizeUp indicates that something can be resized in the upper direction
	ResizeUp
	// ResizeRight indicates that something can be resized in the right direction
	ResizeRight
	// ResizeDown indicates that something can be resized in the downward direction
	ResizeDown
	// ResizeLeft indicates that something can be resized in the left direction
	ResizeLeft
	// ResizeN indicates that something can be resized in the upper direction
	ResizeN
	// ResizeE indicates that something can be resized in the right direction
	ResizeE
	// ResizeS indicates that something can be resized in the downward direction
	ResizeS
	// ResizeW indicates that something can be resized in the left direction
	ResizeW
	// ResizeNE indicates that something can be resized in the upper-right direction
	ResizeNE
	// ResizeNW indicates that something can be resized in the upper-left direction
	ResizeNW
	// ResizeSE indicates that something can be resized in the lower-right direction
	ResizeSE
	// ResizeSW indicates that something can be resized in the lower-left direction
	ResizeSW
	// ResizeEW indicates that something can be resized bidirectionally in the right-left direction
	ResizeEW
	// ResizeNS indicates that something can be resized bidirectionally in the top-bottom direction
	ResizeNS
	// ResizeNESW indicates that something can be resized bidirectionally in the top-right to bottom-left direction
	ResizeNESW
	// ResizeNWSE indicates that something can be resized bidirectionally in the top-left to bottom-right direction
	ResizeNWSE
	// ZoomIn indicates that something can be zoomed in
	ZoomIn
	// ZoomOut indicates that something can be zoomed out
	ZoomOut
	// ScreenshotSelection indicates that a screenshot selection box is being selected
	ScreenshotSelection
	// ScreenshotWindow indicates that a screenshot is being taken of an entire window
	ScreenshotWindow
	// Poof indicates that an item will dissapear when it is released
	Poof
)

// Hotspots contains the cursor hotspot points for every cursor.
// It is initialized to contain the hotspots for all of the default
// cursors, but it should be extended by anyone defining custom cursors.
// Each hotspot is expressed in terms of two times the percentage of the
// size of the cursor it is from the top-left corner (0-200).
var Hotspots = map[enums.Enum]image.Point{
	Arrow:               image.Pt(53, 13),
	ContextMenu:         image.Pt(29, 28),
	Help:                image.Pt(99, 99),
	Pointer:             image.Pt(69, 44),
	Progress:            image.Pt(54, 13),
	Wait:                image.Pt(103, 99),
	Cell:                image.Pt(98, 100),
	Crosshair:           image.Pt(100, 100),
	Text:                image.Pt(99, 103),
	VerticalText:        image.Pt(103, 99),
	Alias:               image.Pt(120, 55),
	Copy:                image.Pt(54, 13),
	Move:                image.Pt(100, 100), // TODO
	NotAllowed:          image.Pt(54, 13),
	Grab:                image.Pt(98, 66),
	Grabbing:            image.Pt(107, 97),
	ResizeCol:           image.Pt(100, 100),
	ResizeRow:           image.Pt(100, 100),
	ResizeUp:            image.Pt(100, 100),
	ResizeRight:         image.Pt(100, 100),
	ResizeDown:          image.Pt(100, 100),
	ResizeLeft:          image.Pt(100, 100),
	ResizeN:             image.Pt(100, 100),
	ResizeE:             image.Pt(100, 100),
	ResizeS:             image.Pt(100, 100),
	ResizeW:             image.Pt(100, 100),
	ResizeNE:            image.Pt(100, 100),
	ResizeNW:            image.Pt(100, 100),
	ResizeSE:            image.Pt(100, 100),
	ResizeSW:            image.Pt(100, 100),
	ResizeEW:            image.Pt(100, 100),
	ResizeNS:            image.Pt(100, 100),
	ResizeNESW:          image.Pt(100, 100),
	ResizeNWSE:          image.Pt(100, 100),
	ZoomIn:              image.Pt(100, 100),
	ZoomOut:             image.Pt(100, 100),
	ScreenshotSelection: image.Pt(100, 100), // TODO
	ScreenshotWindow:    image.Pt(100, 100), // TODO
	Poof:                image.Pt(100, 100), // TODO
}
