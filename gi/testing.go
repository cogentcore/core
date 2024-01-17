// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/grows/images"
)

// AssertRender makes a new window from the scene, waits until it is shown
// and all events have been handled, does any necessary re-rendering,
// asserts that its rendered image is the same as that stored at the given
// filename, saving the image to that filename if it does not already exist,
// and then closes the window. It does not return until all of those steps\
// are completed. If a function is passed for the final argument, it is called
// after the scene is shown.
func (sc *Scene) AssertRender(t images.TestingT, filename string, fun ...func()) {
	showed := make(chan struct{})
	sc.OnShow(func(e events.Event) {
		if len(fun) > 0 {
			fun[0]()
		}
		showed <- struct{}{}
	})
	sc.NewWindow().Run()
	<-showed

	rw := sc.RenderWin()
	rw.NoEventsChan = make(chan struct{})
	<-rw.NoEventsChan
	rw.NoEventsChan = nil

	sc.DoNeedsRender()

	sc.AssertPixels(t, filename)
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
