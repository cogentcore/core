// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build debug

package gimain

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
)

// DebugEnumSizes is a startup function that reports current sizes of some big
// enums, just to make sure everything is well below 64..
func DebugEnumSizes() {
	fmt.Printf("ki.NodeFlagsN: %d\n", ki.FlagsN)
	fmt.Printf("gi.NodeFlagsN: %d\n", gi.NodeFlagsN)
	fmt.Printf("gi.WinFlagN: %d\n", gi.WinFlagN)
	fmt.Printf("gi.VpFlagN: %d\n", gi.VpFlagN)
}
