// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/goosi/events"
	"goki.dev/grows/images"
)

// RunTest runs the given function after calling [driver.Main] and [Init].
// It should only be used in tests, and it should typically be called in
// TestMain. For example:
//
//	func TestMain(m *testing.M) {
//		gi.RunTest(func() {
//			os.Exit(m.Run())
//		})
//	}
//
//	func TestSomething(t *testing.T) {
//		sc := gi.NewScene()
//		gi.NewLabel(sc).SetText("Something")
//		sc.AssertPixelsOnShow(t, "something")
//	}
func RunTest(test func()) {
	driver.Main(func(a goosi.App) {
		Init()
		test()
	})
}

// AssertPixelsOnShow is a helper function that makes a new window from
// the scene, waits until it is shown, calls [Scene.AssertPixels]
// with the given values, and then closes the window.
// It does not return until all of those steps are completed.
// If a function is passed for the final argument, it is called after the
// scene is shown, right before [Scene.AssertPixels] is called. Also,
// if a function is passed, [Scene.DoNeedsRender] is also called before
// [Scene.AssertPixels].
func (sc *Scene) AssertPixelsOnShow(t images.TestingT, filename string, fun ...func()) {
	showed := make(chan struct{})
	sc.OnShow(func(e events.Event) {
		if len(fun) > 0 {
			fun[0]()
			sc.DoNeedsRender()
		}
		sc.AssertPixels(t, filename)
		showed <- struct{}{}
	})
	sc.NewWindow().Run()
	<-showed
	sc.Close()
}

// AssertPixels asserts that [Scene.Pixels] is equivalent
// to the image stored at the given filename in the testdata directory,
// with ".png" added to the filename if there is no extension
// (eg: "button" becomes "testdata/button.png").
// If it is not, it fails the test with an error, but continues its
// execution. If there is no image at the given filename in the testdata
// directory, it creates the image.
func (sc *Scene) AssertPixels(t images.TestingT, filename string) {
	images.Assert(t, sc.Pixels, filename)
}
