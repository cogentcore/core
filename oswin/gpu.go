// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package oswin

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
	UseContext(win Window)

	// ClearContext unsets the current OpenGL context for given window
	// and unlocks the per-window mutex.
	// Assumes that UseContext was previously called on window.
	ClearContext(win Window)

	// NewProgram returns a new program with the given vertex and fragment
	// shader programs, in GLSL.  return value is the GL handle.
	NewProgram(vertexShaderSrc, fragmentShaderSrc string) (uint32, error)

	// CompileShader returns a new compiled shader from given source.
	// return value is the GL handle.
	CompileShader(source string) (uint32, error)
}
