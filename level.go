// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grog

//go:generate enumgen

// Level represents a level of logging or printing verbosity.
// It can be used by both users and developers to determine
// the level of verbosity when running commands and logging.
// The user preference verbosity level is stored in [UserLevel].
type Level int //enums:enum

const (
	// Debug indicates that a message is a debugging message,
	// or to show all messages in the context of debugging.
	// It can be set by the end user as the value of [UserLevel]
	// through the "-vv" (very verbose) flag in xe.
	Debug Level = -4

	// Info indicates that a message is an informational message,
	// or to show all messages at or above the info level.
	// It can be set by the end user as the value of [UserLevel]
	// through the "-v" (verbose) flag in xe.
	Info Level = 0

	// Warn indicates that a message is a warning message,
	// or to show all messages at or above the warning level.
	// It is the default value for [UserLevel].
	Warn Level = 4

	// Error indicates that a message is an error message,
	// or to only show error messages. It can be set by the
	// end user as the value of [UserLevel] through the "-q"
	// (quiet) flag in xe.
	Error Level = 8
)
