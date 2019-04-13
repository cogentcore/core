// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"sync"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/glgpu"
	"github.com/goki/gi/oswin/gpu"
	"golang.org/x/image/math/f64"
)

// how to code opengl to be vulkan-friendly
// https://developer.nvidia.com/opengl-vulkan
// https://www.slideshare.net/CassEveritt/approaching-zero-driver-overhead
// in general, use drawelements instead of arrays (i.e., use indexing)

var glTypes = map[gpu.Types]int32{
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
	mu sync.Mutex
}

var theGPU = &gpuImpl{}

func (gpu *gpuImpl) UseContext(win oswin.Window) {
	w := win.(*windowImpl)
	w.glctxMu.Lock()
	w.glw.MakeContextCurrent()
}

func (gpu *gpuImpl) ClearContext(win oswin.Window) {
	w := win.(*windowImpl)
	glfw.DetachCurrentContext()
	w.glctxMu.Unlock()
}

func (gpu *gpuImpl) Type(typ gpu.Types) int32 {
	return glTypes[typ]
}

// NewProgram returns a new Program with given name -- for standalone programs.
// See also NewPipeline.
func (gpu *gpuImpl) NewProgram(name string) gpu.Program {
	pr := &glgpu.Program{}
	pr.SetName(name)
	return pr
}

// NewPipeline returns a new Pipeline to manage multiple coordinated Programs.
func (gpu *gpuImpl) NewPipeline(name string) gpu.Pipeline {
	pl := &glgpu.Pipeline{}
	pl.SetName(name)
	return pr
}

// NewBufferMgr returns a new BufferMgr for managing Vectors and Indexes for rendering.
func (gpu *gpuImpl) NewBufferMgr() gpu.BufferMgr {
	bm := &glgpu.BufferMgr{}
	return bm
}

// 	NextUniformBindingPoint returns the next avail uniform binding point.
// Counts up from 0 -- this call increments for next call.
func (gpu *gpuImpl) NextUniformBindingPoint() int {

}

func writeAff3(u int32, a f64.Aff3) {
	var m [9]float32
	m[0*3+0] = float32(a[0*3+0])
	m[0*3+1] = float32(a[1*3+0])
	m[0*3+2] = 0
	m[1*3+0] = float32(a[0*3+1])
	m[1*3+1] = float32(a[1*3+1])
	m[1*3+2] = 0
	m[2*3+0] = float32(a[0*3+2])
	m[2*3+1] = float32(a[1*3+2])
	m[2*3+2] = 1
	gl.UniformMatrix3fv(u, 1, false, &m[0])
	glErrProc("writeaff3")
}
