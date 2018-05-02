// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin !386,!amd64 ios
// +build !linux android
// +build !windows
// +build !openbsd

package gldriver

import (
	"fmt"
	"runtime"

	"github.com/goki/goki/gi/oswin"
)

func newWindow(opts *oswin.NewWindowOptions) (uintptr, error) { return 0, nil }

func initWindow(id *windowImpl) {}
func showWindow(id *windowImpl) {}
func closeWindow(id uintptr)    {}
func drawLoop(w *windowImpl)    {}

func main(f func(oswin.Screen)) error {
	return fmt.Errorf("gldriver: unsupported GOOS/GOARCH %s/%s", runtime.GOOS, runtime.GOARCH)
}
