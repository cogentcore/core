// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"image"
)

// TheGPU is the current oswin GPU instance
var TheGPU GPU

// GPU provides the main interface to the GPU hardware.
// Currently based on OpenGL.
// All calls apply to the current context, which must be set with
// Activate() call on relevant oswin.Window.  Framebuffer.Activate() will
// also set the rendering target to a framebuffer instead of the window.
// Furthermore, all GPU calls must be embedded in oswin.TheApp.RunOnMain
// function call to run on the main thread:
//
// oswin.TheApp.RunOnMain(func() {
//    win.Activate()
//    // do GPU calls here
// })
//
type GPU interface {
	// Init initializes the GPU framework etc
	// if debug is true, then it turns on debugging mode
	// and, if available, enables automatic error callback
	// unfortunately that is not avail for OpenGL on mac
	// and possibly other systems, so ErrCheck must be used
	// but it is a NOP if the callback method is avail.
	Init(debug bool) error

	// ActivateShared activates the invisible shared context
	// which is shared across all other window / offscreen
	// rendering contexts, and should be used as the context
	// for initializing shared resources.
	ActivateShared() error

	// IsDebug returns true if debug mode is on
	IsDebug() bool

	// ErrCheck checks if there have been any GPU-related errors
	// since the last call to ErrCheck -- if callback errors
	// are avail, then returns most recent such error, which are
	// also automatically logged when they occur.
	ErrCheck(ctxt string) error

	// RenderToWindow sets the current context's window as the
	// render target (i.e., the default framebuffer 0).
	// This can be used if a Framebuffer was previously active.
	// Automatically called during UseContext to make sure.
	RenderToWindow()

	// Type returns the GPU data type id for given type
	Type(typ Types) uint32

	// NewProgram returns a new Program with given name -- for standalone programs.
	// See also NewPipeline.
	NewProgram(name string) Program

	// NewPipeline returns a new Pipeline to manage multiple coordinated Programs.
	NewPipeline(name string) Pipeline

	// NewBufferMgr returns a new BufferMgr for managing Vectors and Indexes for rendering.
	NewBufferMgr() BufferMgr

	// NewInputVectors returns a new Vectors input variable that has a pre-specified
	// layout(location = X) in programs -- allows same inputs to be used across a set
	// of programs that all use the same locations.
	NewInputVectors(name string, loc int, typ VectorType, role VectorRoles) Vectors

	// NewTexture2D returns a new Texture2D with given name (optional).
	// These Texture2D's must be Activate()'d and Delete()'d and otherwise managed
	// (no further tracking is done by the gpu framework)
	NewTexture2D(name string) Texture2D

	// NewFramebuffer returns a new Framebuffer for rendering directly
	// onto a texture instead of onto the Window (i.e., for offscreen rendering).
	// samples is typically 4 for multisampling anti-aliasing (generally recommended).
	// See also Texture2D.ActivateFramebuffer to activate a framebuffer for rendering
	// to an existing texture.
	NewFramebuffer(name string, size image.Point, samples int) Framebuffer

	// NewUniforms makes a new named set of uniforms (i.e,. a Uniform Buffer Object)
	// These uniforms can be bound to programs -- first add all the uniform variables
	// and then AddUniforms to each program that uses it.
	// Uniforms will be bound etc when the program is compiled.
	NewUniforms(name string) Uniforms

	// 	NextUniformBindingPoint returns the next avail uniform binding point.
	// Counts up from 0 -- this call increments for next call.
	NextUniformBindingPoint() int
}
