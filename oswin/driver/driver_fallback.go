// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build !darwin
// +build !linux android
// +build !windows
// +build !dragonfly
// +build !openbsd
// +build !3d

package driver

import (
	"errors"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/errscreen"
)

func main(f func(oswin.App)) {
	f(errscreen.Stub(errors.New("no driver for accessing a screen")))
}
