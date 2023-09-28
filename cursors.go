// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package cursors provides Go constant names for cursors as SVG files.
package cursors

import "embed"

// Cursors contains all of the default embedded svg cursors
//
//go:embed svg/*.svg
var Cursors embed.FS

// Cursor represents a cursor
type Cursor int32 //enums:enum

// Cursor constants, with names based on CSS (see https://developer.mozilla.org/en-US/docs/Web/CSS/cursor)
// TODO: maybe add NoDrop and AllScroll
const (
	// Default is a default arrow cursor
	Default Cursor = iota
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
