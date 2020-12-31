// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"fmt"
	"image"
	"log"
	"sync"
	"unsafe"

	"github.com/go-gl/gl/v3.3-core/gl"
	"github.com/goki/gi/oswin/driver/internal/glgpu"
	"github.com/goki/gi/oswin/gpu"
)

// how to code opengl to be vulkan-friendly
// https://developer.nvidia.com/opengl-vulkan
// https://www.slideshare.net/CassEveritt/approaching-zero-driver-overhead
// in general, use drawelements instead of arrays (i.e., use indexing)

var glTypes = map[gpu.Types]uint32{
	gpu.UndefType: gl.FLOAT,
	gpu.Bool:      gl.BOOL,
	gpu.Int:       gl.INT,
	gpu.UInt:      gl.UNSIGNED_INT,
	gpu.Float32:   gl.FLOAT,
	gpu.Float64:   gl.DOUBLE,
}

type gpuImpl struct {
	// mu is in case we need a cpu-wide mutex -- mostly it is the
	// window-specific glctxtMu that is used
	mu        sync.Mutex
	bindPoint int
	debug     bool
	dbCbSet   bool  // is the callback debug set?
	lastErr   error // last error from callback
}

var theGPU = &gpuImpl{}

// Init initializes the GPU framework etc
// if debug is true, then it turns on debugging mode
// and, if available, enables automatic error callback
// unfortunately that is not avail for OpenGL on mac
// and possibly other systems, so ErrCheck must be used
// but it is a NOP if the callback method is avail.
func (gp *gpuImpl) Init(debug bool) error {
	if err := gl.Init(); err != nil {
		return err
	}
	gpu.TheGPU = theGPU
	gp.debug = debug
	if debug {
		version := gl.GoStr(gl.GetString(gl.VERSION))
		fmt.Println("OpenGL version", version)
		// Query the extensions to determine if we can enable the debug callback
		var numExtensions int32
		gl.GetIntegerv(gl.NUM_EXTENSIONS, &numExtensions)
		for i := int32(0); i < numExtensions; i++ {
			extension := gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i)))
			// fmt.Println(extension)
			if extension == "GL_ARB_debug_output" {
				gp.dbCbSet = true
				gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS_ARB)
				gl.DebugMessageCallbackARB(gl.DebugProc(gp.DebugMsg), gl.Ptr(nil))
			}
		}
	}
	return nil
}

// ActivateShared activates the invisible shared context
// which is shared across all other window / offscreen
// rendering contexts, and should be used as the context
// for initializing shared resources.
func (gp *gpuImpl) ActivateShared() error {
	if theApp.shareWin == nil {
		err := fmt.Errorf("glos GPU.ActivateShared -- gl not yet Initialized and shareWin is nil")
		log.Println(err)
		return err
	}
	theApp.shareWin.MakeContextCurrent()
	return nil
}

// IsDebug returns true if debug mode is on
func (gp *gpuImpl) IsDebug() bool {
	return gp.debug
}

// ErrCheck checks if there have been any GPU-related errors
// since the last call to ErrCheck -- if callback errors
// are avail, then returns most recent such error, which are
// also automatically logged when they occur.
func (gp *gpuImpl) ErrCheck(ctxt string) error {
	var err error
	if gp.dbCbSet {
		err = gp.lastErr
		gp.lastErr = nil
		return err
	}
	for {
		glerr := gl.GetError()
		if glerr == gl.NO_ERROR {
			break
		}
		errstr, _ := glErrStrings[glerr]
		err = fmt.Errorf("glos gl error in context: %s:\n\t%x = %s", ctxt, glerr, errstr)
		// log.Println(err)
	}
	gp.lastErr = err
	return err
}

func (gp *gpuImpl) RenderToWindow() {
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (gp *gpuImpl) Type(typ gpu.Types) uint32 {
	return glTypes[typ]
}

// NewProgram returns a new Program with given name -- for standalone programs.
// See also NewPipeline.
func (gp *gpuImpl) NewProgram(name string) gpu.Program {
	pr := &glgpu.Program{}
	pr.SetName(name)
	return pr
}

// NewPipeline returns a new Pipeline to manage multiple coordinated Programs.
func (gp *gpuImpl) NewPipeline(name string) gpu.Pipeline {
	pl := &glgpu.Pipeline{}
	pl.SetName(name)
	return pl
}

// NewBufferMgr returns a new BufferMgr for managing Vectors and Indexes for rendering.
func (gp *gpuImpl) NewBufferMgr() gpu.BufferMgr {
	bm := &glgpu.BufferMgr{}
	return bm
}

// NewInputVectors returns a new Vectors input variable that has a pre-specified
// layout(location = X) in programs -- allows same inputs to be used across a set
// of programs that all use the same locations.
func (gp *gpuImpl) NewInputVectors(name string, loc int, typ gpu.VectorType, role gpu.VectorRoles) gpu.Vectors {
	v := &glgpu.Vectors{}
	v.Set(name, uint32(loc), typ, role)
	return v
}

// NewTexture2D returns a new Texture2D with given name (optional).
// These Texture2D's must be Activate()'d and Delete()'d and otherwise managed
// (no further tracking is done by the gpu framework)
func (gp *gpuImpl) NewTexture2D(name string) gpu.Texture2D {
	tx := &textureImpl{name: name}
	return tx
}

// NewFramebuffer returns a new Framebuffer for rendering directly
// onto a texture instead of onto the Window (i.e., for offscreen rendering).
// samples is typically 4 for multisampling anti-aliasing (generally recommended).
// See also Texture2D.ActivateFramebuffer to activate a framebuffer for rendering
// to an existing texture.
func (gp *gpuImpl) NewFramebuffer(name string, size image.Point, samples int) gpu.Framebuffer {
	fb := &glgpu.Framebuffer{}
	fb.SetName(name)
	fb.SetSize(size)
	fb.SetSamples(samples)
	return fb
}

// NewUniforms makes a new named set of uniforms (i.e,. a Uniform Buffer Object)
// These uniforms can be bound to programs -- first add all the uniform variables
// and then AddUniforms to each program that uses it.
// Uniforms will be bound etc when the program is compiled.
func (gp *gpuImpl) NewUniforms(name string) gpu.Uniforms {
	us := &glgpu.Uniforms{}
	us.SetName(name)
	return us
}

// 	NextUniformBindingPoint returns the next avail uniform binding point.
// Counts up from 0 -- this call increments for next call.
func (gp *gpuImpl) NextUniformBindingPoint() int {
	bp := gp.bindPoint
	gp.bindPoint++
	return bp
}

//////////////////////////////////////////////////////////////
// Debugging

func (gp *gpuImpl) DebugMsg(
	source uint32,
	gltype uint32,
	id uint32,
	severity uint32,
	length int32,
	message string,
	userParam unsafe.Pointer) {
	gp.lastErr = fmt.Errorf("glos gl error msg: %v source: %v gltype %v id %v severity %v length %v",
		message, source, gltype, id, severity, length)
	log.Println(gp.lastErr)
}

var glErrStrings = map[uint32]string{
	gl.INVALID_ENUM:                  "INVALID_ENUM: Given when an enumeration parameter is not a legal enumeration for that function. This is given only for local problems; if the spec allows the enumeration in certain circumstances, where other parameters or state dictate those circumstances, then GL_INVALID_OPERATION is the result instead.",
	gl.INVALID_VALUE:                 "INVALID_VALUE: Given when a value parameter is not a legal value for that function. This is only given for local problems; if the spec allows the value in certain circumstances, where other parameters or state dictate those circumstances, then GL_INVALID_OPERATION is the result instead.",
	gl.INVALID_OPERATION:             "INVALID_OPERATION: Given when the set of state for a command is not legal for the parameters given to that command. It is also given for commands where combinations of parameters define what the legal parameters are.",
	gl.STACK_OVERFLOW:                "STACK_OVERFLOW: Given when a stack pushing operation cannot be done because it would overflow the limit of that stack's size.",
	gl.STACK_UNDERFLOW:               "STACK_UNDERFLOW: Given when a stack popping operation cannot be done because the stack is already at its lowest point.",
	gl.OUT_OF_MEMORY:                 "OUT_OF_MEMORY: Given when performing an operation that can allocate memory, and the memory cannot be allocated. The results of OpenGL functions that return this error are undefined; it is allowable for partial operations to happen.",
	gl.INVALID_FRAMEBUFFER_OPERATION: "INVALID_FRAMEBUFFER_OPERATION: Given when doing anything that would attempt to read from or write/render to a framebuffer that is not complete.",
}
