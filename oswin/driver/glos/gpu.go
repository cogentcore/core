// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package glos

import (
	"fmt"
	"log"
	"sync"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
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
	gpu.UInt:      gl.UINT,
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
		log.Println(err)
	}
	gp.lastErr = err
	return err
}

func (gp *gpuImpl) UseContext(win oswin.Window) {
	w := win.(*windowImpl)
	w.glctxMu.Lock()
	w.glw.MakeContextCurrent()
}

func (gp *gpuImpl) ClearContext(win oswin.Window) {
	w := win.(*windowImpl)
	glfw.DetachCurrentContext()
	w.glctxMu.Unlock()
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
