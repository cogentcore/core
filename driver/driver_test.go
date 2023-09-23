// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package driver

import (
	"fmt"
	"testing"

	"goki.dev/goosi"
)

func TestMain(t *testing.T) {
	Main(func(a goosi.App) {
		fmt.Println(goosi.TheApp.Name())
	})
}
