// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build darwin
// +build !3d

package macdriver

import (
	"runtime"

	"golang.org/x/mobile/gl"
)

// NewContext creates an OpenGL ES context with a dedicated processing thread.
func NewContext() (gl.Context, error) {
	glctx, worker := gl.NewContext()

	errCh := make(chan error)
	workAvailable := worker.WorkAvailable()
	go func() {
		runtime.LockOSThread()
		err := surfaceCreate()
		errCh <- err
		if err != nil {
			return
		}

		for range workAvailable {
			worker.DoWork()
		}
	}()
	if err := <-errCh; err != nil {
		return nil, err
	}
	return glctx, nil
}
