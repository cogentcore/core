// Copyright 2019 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d

package glos

import (
	"fmt"
	"strings"
	"sync"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/goki/gi/oswin"
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

func (gpu *gpuImpl) NewProgram(vertexSrc, fragmentSrc string) (uint32, error) {
	vertexShader, err := gpu.CompileShader(vertexSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := gpu.CompileShader(fragmentSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)
	if glosDebug {
		gl.ValidateProgram(program)
	}

	gl.DetachShader(program, vertexShader)
	gl.DetachShader(program, fragmentShader)
	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	return program, nil
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
