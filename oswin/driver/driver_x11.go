// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd

package driver

import (
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/driver/x11driver"
)

func main(f func(oswin.App)) {
	x11driver.Main(f)
}
