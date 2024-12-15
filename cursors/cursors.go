// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursors provides Go constant names for cursors as SVG files.
package cursors

//go:generate core generate

import (
	"embed"
	"image"

	"cogentcore.org/core/enums"
)

// Cursors contains all of the default embedded svg cursors.
//
//go:embed svg/*.svg
var Cursors embed.FS

// Cursor represents a cursor.
type Cursor int32 //enums:enum -transform kebab

// TODO: maybe add NoDrop and AllScroll

// Cursor constants, with names based on CSS (see https://developer.mozilla.org/en-US/docs/Web/CSS/cursor).
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
// Each hotspot is expressed as a point relative to the top-left corner
// of the cursor, on a scale of 0-256, which is scaled to the size of
// the cursor later on.
var Hotspots = map[enums.Enum]image.Point{
	Arrow:               image.Pt(88, 80),
	ContextMenu:         image.Pt(72, 80),
	Help:                image.Pt(128, 128),
	Pointer:             image.Pt(104, 76),
	Progress:            image.Pt(64, 24),
	Wait:                image.Pt(132, 127),
	Cell:                image.Pt(125, 128),
	Crosshair:           image.Pt(128, 128),
	Text:                image.Pt(128, 128),
	VerticalText:        image.Pt(128, 124),
	Alias:               image.Pt(156, 80),
	Copy:                image.Pt(64, 24),
	Move:                image.Pt(128, 128),
	NotAllowed:          image.Pt(64, 24),
	Grab:                image.Pt(124, 124),
	Grabbing:            image.Pt(124, 124),
	ResizeCol:           image.Pt(128, 128),
	ResizeRow:           image.Pt(128, 128),
	ResizeUp:            image.Pt(128, 128),
	ResizeRight:         image.Pt(128, 128),
	ResizeDown:          image.Pt(128, 128),
	ResizeLeft:          image.Pt(128, 128),
	ResizeN:             image.Pt(128, 128),
	ResizeE:             image.Pt(128, 128),
	ResizeS:             image.Pt(128, 128),
	ResizeW:             image.Pt(128, 128),
	ResizeNE:            image.Pt(128, 128),
	ResizeNW:            image.Pt(128, 128),
	ResizeSE:            image.Pt(128, 128),
	ResizeSW:            image.Pt(128, 128),
	ResizeEW:            image.Pt(128, 128),
	ResizeNS:            image.Pt(128, 128),
	ResizeNESW:          image.Pt(128, 128),
	ResizeNWSE:          image.Pt(128, 128),
	ZoomIn:              image.Pt(128, 128),
	ZoomOut:             image.Pt(128, 128),
	ScreenshotSelection: image.Pt(128, 128),
	ScreenshotWindow:    image.Pt(128, 128),
	Poof:                image.Pt(64, 24),
}
