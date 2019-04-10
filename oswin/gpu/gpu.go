// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/gi/oswin"

// TheGPU is the current oswin GPU instance
var TheGPU GPU

// GPU represents provides the main interface to the GPU hardware.
// currently based on OpenGL.
// All calls apply to the current context, which must be set with
// UseContext call, and cleared after with ClearContext, in strict
// pairing as there is a mutex locked and unlocked with each call.
type GPU interface {
	// UseContext sets the current OpenGL context to be that of given window.
	// All methods in GPU operate on the current context.
	// Also locks a per-window mutex, as GL calls are not threadsafe -- MUST
	// call ClearContext after every call to UseContext
	UseContext(win oswin.Window)

	// ClearContext unsets the current OpenGL context for given window
	// and unlocks the per-window mutex.
	// Assumes that UseContext was previously called on window.
	ClearContext(win oswin.Window)

	// NewProgram returns a new Program with given name -- for standalone programs.
	// See also NewPipeline.
	NewProgram(name string) Program

	// NewPipeline returns a new Pipeline to manage multiple coordinated Programs.
	NewPipeline(name string) Pipeline

	// NewBufferMgr returns a new BufferMgr for managing Vectors and Indexes for rendering.
	NewBufferMgr() BufferMgr

	// 	NextUniformBindingPoint returns the next avail uniform binding point.
	// Counts up from 0 -- this call increments for next call.
	NextUniformBindingPoint() int32
}
