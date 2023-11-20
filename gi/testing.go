// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
)

// RunTest is a simple helper function that runs the given
// function after calling [driver.Main] and [Init]. It should
// only be used in tests. For example:
//
//	func TestSomething(t *testing.T) {
//		gi.RunTest(func() {
//			sc := gi.NewScene()
//			gi.NewLabel(sc).SetText("Something")
//			gi.NewWindow(sc).Run()
//			goosi.AssertCaptureIs(t, "something")
//		})
//	}
func RunTest(test func()) {
	driver.Main(func(a goosi.App) {
		Init()
		test()
	})
}
