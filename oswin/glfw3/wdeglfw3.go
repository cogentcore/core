/*
   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package glfw3

import (
	//	"fmt"
	"github.com/go-gl/gl"
	glfw "github.com/grd/glfw3"
	"github.com/rcoreilly/goki/gi/oswin"
	"image"
	"os"
)

func init() {
	oswin.BackendNewWindow = func(width, height int) (w oswin.OSWindow, err error) {
		var window *OSWindow
		window, err = NewOSWindow(width, height)
		w = window
		return
	}

	oswin.BackendRun = glfw.Main

	oswin.BackendStop = func() {
		glfw.Terminate()
		os.Exit(0)
	}

	go doRun()

	// Don't show the window context before using the Show function.
	go glfw.WindowHint(glfw.Visible, glfw.False)

	go flushBuffer()

	go setGlyph()
}

func doRun() {
	for {
		// Poll for and process events
		glfw.PollEvents()
	}
}

type OSWindow struct {
	win    *glfw.Window
	buffer WinImage
	events chan interface{}
}

var windowMap = make(map[uintptr]*OSWindow)

func NewOSWindow(width, height int) (w *OSWindow, err error) {

	w = new(OSWindow)

	w.win, err = glfw.CreateWindow(width, height, "", nil, nil)
	if err != nil {
		return nil, err
	}

	windowMap[w.win.C()] = w

	w.buffer.RGBA = image.NewRGBA(image.Rect(0, 0, width, height))

	// Events and callback functions for events

	w.events = make(chan interface{})
	w.win.SetMouseButtonCallback(mouseButtonCallback)
	w.win.SetCursorEnterCallback(cursorEnterCallback)
	w.win.SetCursorPositionCallback(cursorPositionCallback)
	w.win.SetFramebufferSizeCallback(framebufferSizeCallback)
	w.win.SetCharacterCallback(characterCallback)
	w.win.SetKeyCallback(keyCallback)
	w.checkShouldClose()

	return
}

func (w *OSWindow) SetTitle(title string) {
	w.win.SetTitle(title)
}

func (w *OSWindow) SetCursor(cursor oswin.Cursor) {
	// TODO
}

func (w *OSWindow) SetSize(width, height int) {
	w.win.SetSize(width, height)
}

func (w *OSWindow) Size() (width, height int) {
	return w.win.GetSize()
}

func (w *OSWindow) LockSize(lock bool) {
	// glfw supports only window size locking when the
	// parameter is set before the creation of the window.
	// It doesn't work on an existing window.
	if lock {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	}
}

func (w *OSWindow) Show() {
	w.win.Show()
}

func (w *OSWindow) Screen() oswin.WinImage {
	return w.buffer
}

func (w *OSWindow) FlushImage(bounds ...image.Rectangle) {

	if w.win.ShouldClose() {
		return
	}

	// TODO: Howto implement ...image.Rectangle

	// flush the buffer
	windowFlushBuffer <- w

	// waiting for the flushing is done before filling the buffer again
	<-windowFlushBufferDone
}

func (w *OSWindow) Close() (err error) {
	w.win.Destroy()
	return
}

func (w *OSWindow) checkShouldClose() {
	go func() {
		for {
			if w.win.ShouldClose() {
				var cev oswin.CloseEvent
				w.events <- cev
				break
			}
		}
	}()
}

type WinImage struct {
	*image.RGBA
}

func (buffer WinImage) CopyRGBA(src *image.RGBA, r image.Rectangle) {
	// clip r against each image's bounds and move sp accordingly (see draw.clip())
	sp := image.ZP
	orig := r.Min
	r = r.Intersect(buffer.Bounds())
	r = r.Intersect(src.Bounds().Add(orig.Sub(sp)))
	dx := r.Min.X - orig.X
	dy := r.Min.Y - orig.Y
	(sp).X += dx
	(sp).Y += dy

	i0 := (r.Min.X - buffer.Rect.Min.X) * 4
	i1 := (r.Max.X - buffer.Rect.Min.X) * 4
	si0 := (sp.X - src.Rect.Min.X) * 4
	yMax := r.Max.Y - buffer.Rect.Min.Y

	y := r.Min.Y - buffer.Rect.Min.Y
	sy := sp.Y - src.Rect.Min.Y
	for ; y != yMax; y, sy = y+1, sy+1 {
		dpix := buffer.Pix[y*buffer.Stride:]
		spix := src.Pix[sy*src.Stride:]

		for i, si := i0, si0; i < i1; i, si = i+4, si+4 {
			dpix[i+0] = spix[si+2]
			dpix[i+1] = spix[si+1]
			dpix[i+2] = spix[si+0]
			dpix[i+3] = spix[si+3]
		}
	}
}

var (
	windowFlushBuffer     = make(chan *OSWindow)
	windowFlushBufferDone = make(chan bool)
)

func flushBuffer() {
	for {

		w := <-windowFlushBuffer

		w.win.MakeContextCurrent()

		imgWidth := w.buffer.Rect.Max.X
		imgHeight := w.buffer.Rect.Max.Y

		windowWidth, windowHeight := w.Size()

		// gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.RasterPos2f(-1, 1)
		gl.PixelZoom(1, -1)
		gl.Viewport(0, 0, windowWidth, windowHeight)
		gl.DrawPixels(imgWidth, imgHeight, gl.RGBA, gl.UNSIGNED_BYTE, &w.buffer.Pix[0])

		w.win.SwapBuffers()

		windowFlushBufferDone <- true
	}
}
