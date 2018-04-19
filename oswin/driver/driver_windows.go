// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package driver

import (
	"github.com/rcoreilly/goki/gi/oswin/driver/windriver"
)

func main(f func(oswin.Ap)) {
	windriver.Main(f)
}
