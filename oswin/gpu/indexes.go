// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import "github.com/goki/mat32"

// IndexesBuffer manages a buffer of indexes for index-based rendering
// (i.e., GL_ELEMENT_ARRAY_BUFFER for glDrawElements calls in OpenGL).
type IndexesBuffer interface {
	// SetLen sets the number of indexes in buffer
	SetLen(ln int)

	// Len returns the number of indexes in bufer
	Len() int

	// Set sets the indexes by copying given data
	Set(idxs mat32.ArrayU32)

	// Returns the indexes (direct copy of internal buffer -- can be modified)
	Indexes() mat32.ArrayU32

	// Activate binds buffer as active one
	Activate()

	// Handle returns the unique handle for this buffer -- only valid after Activate()
	Handle() uint32

	// Transfer transfers data to GPU -- Activate must have been called with no other
	// such buffers activated in between.  Automatically uses re-specification
	// strategy per: https://www.khronos.org/opengl/wiki/Buffer_Object_Streaming
	// so it is safe if buffer was still being used from prior GL rendering call.
	Transfer()

	// Delete deletes the GPU resources associated with this buffer
	// (requires Activate to re-establish a new one).
	// Should be called prior to Go object being deleted
	// (ref counting can be done externally).
	Delete()
}
