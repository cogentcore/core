// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"github.com/goki/gi/oswin"
)

// TheGPU is the current oswin GPU instance
var TheGPU GPU

// GPU provides the main interface to the GPU hardware.
// currently based on OpenGL.
// All calls apply to the current context, which must be set with
// UseContext call, and cleared after with ClearContext, in strict
// pairing as there is a mutex locked and unlocked with each call.
type GPU interface {
	// Init initializes the GPU framework etc
	// if debug is true, then it turns on debugging mode
	// and, if available, enables automatic error callback
	// unfortunately that is not avail for OpenGL on mac
	// and possibly other systems, so ErrCheck must be used
	// but it is a NOP if the callback method is avail.
	Init(debug bool) error

	// IsDebug returns true if debug mode is on
	IsDebug() bool

	// ErrCheck checks if there have been any GPU-related errors
	// since the last call to ErrCheck -- if callback errors
	// are avail, then returns most recent such error, which are
	// also automatically logged when they occur.
	ErrCheck(ctxt string) error

	// UseContext sets the current OpenGL context to be that of given window.
	// All methods in GPU operate on the current context.
	// Also locks a per-window mutex, as GL calls are not threadsafe -- MUST
	// call ClearContext after every call to UseContext
	UseContext(win oswin.Window)

	// ClearContext unsets the current OpenGL context for given window
	// and unlocks the per-window mutex.
	// Assumes that UseContext was previously called on window.
	ClearContext(win oswin.Window)

	// Type returns the GPU data type id for given type
	Type(typ Types) uint32

	// NewProgram returns a new Program with given name -- for standalone programs.
	// See also NewPipeline.
	NewProgram(name string) Program

	// NewPipeline returns a new Pipeline to manage multiple coordinated Programs.
	NewPipeline(name string) Pipeline

	// NewBufferMgr returns a new BufferMgr for managing Vectors and Indexes for rendering.
	NewBufferMgr() BufferMgr

	// 	NextUniformBindingPoint returns the next avail uniform binding point.
	// Counts up from 0 -- this call increments for next call.
	NextUniformBindingPoint() int
}
