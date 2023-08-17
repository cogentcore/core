// Copyright (c) 2021, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cam02

import (
	"fmt"
	"testing"
)

func TestLumAdapt(t *testing.T) {
	fl := LuminanceAdapt(200)
	fmt.Println(fl)
}
