// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !(android || ios)

package driver

import (
	"goki.dev/gi/v2/oswin"
	"goki.dev/gi/v2/oswin/driver/vkos"
)

func driverMain(f func(oswin.App)) {
	vkos.Main(f)
}
