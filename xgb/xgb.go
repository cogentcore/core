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

package xgb

import (
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/ewmh"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/BurntSushi/xgbutil/xwindow"
	"github.com/rcoreilly/goki/gi"
	"image"
	"os"
	"sync"
)

func init() {
	gi.BackendNewWindow = func(width, height int) (w gi.OSWindow, err error) {
		w, err = NewOSWindow(width, height)
		return
	}
	ch := make(chan struct{}, 1)
	gi.BackendRun = func() {
		<-ch
	}
	gi.BackendStop = func() {
		ch <- struct{}{}
	}
}

const AllEventsMask = xproto.EventMaskKeyPress |
	xproto.EventMaskKeyRelease |
	xproto.EventMaskKeymapState |
	xproto.EventMaskButtonPress |
	xproto.EventMaskButtonRelease |
	xproto.EventMaskEnterWindow |
	xproto.EventMaskLeaveWindow |
	xproto.EventMaskPointerMotion |
	xproto.EventMaskStructureNotify

type OSWindow struct {
	win           *xwindow.OSWindow
	xu            *xgbutil.XUtil
	conn          *xgb.Conn
	buffer        *xgraphics.Image
	bufferLck     *sync.Mutex
	width, height int
	lockedSize    bool
	closed        bool
	cursor        gi.Cursor // most recently set cursor

	events chan interface{}
}

func NewOSWindow(width, height int) (w *OSWindow, err error) {

	w = new(OSWindow)
	w.width, w.height = width, height

	w.xu, err = xgbutil.NewConn()
	if err != nil {
		return
	}

	w.conn = w.xu.Conn()
	screen := w.xu.Screen()

	w.win, err = xwindow.Generate(w.xu)
	if err != nil {
		return
	}

	err = w.win.CreateChecked(screen.Root, 600, 500, width, height, 0)
	if err != nil {
		return
	}

	w.win.Listen(AllEventsMask)

	err = icccm.WmProtocolsSet(w.xu, w.win.Id, []string{"WM_DELETE_WINDOW"})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		err = nil
	}

	w.bufferLck = &sync.Mutex{}
	w.buffer = xgraphics.New(w.xu, image.Rect(0, 0, width, height))
	w.buffer.XSurfaceSet(w.win.Id)

	keyMap, modMap := keybind.MapsGet(w.xu)
	keybind.KeyMapSet(w.xu, keyMap)
	keybind.ModMapSet(w.xu, modMap)

	w.events = make(chan interface{})

	w.SetIcon(Gordon)
	w.SetIconName("Go")

	go w.handleEvents()

	return
}

func (w *OSWindow) SetTitle(title string) {
	if w.closed {
		return
	}
	err := ewmh.WmNameSet(w.xu, w.win.Id, title)
	if err != nil {
		// TODO: log
	}
	return
}

func (w *OSWindow) SetSize(width, height int) {
	if w.closed {
		return
	}

	w.width, w.height = width, height
	if w.lockedSize {
		w.updateSizeHints()
	}
	w.win.Resize(width, height)
	return
}

func (w *OSWindow) Size() (width, height int) {
	if w.closed {
		return
	}
	width, height = w.width, w.height
	return
}

func (w *OSWindow) LockSize(lock bool) {
	w.lockedSize = lock
	w.updateSizeHints()
}

func (w *OSWindow) updateSizeHints() {
	hints := new(icccm.NormalHints)
	if w.lockedSize {
		hints.Flags = icccm.SizeHintPMinSize | icccm.SizeHintPMaxSize
		hints.MinWidth = uint(w.width)
		hints.MaxWidth = uint(w.width)
		hints.MinHeight = uint(w.height)
		hints.MaxHeight = uint(w.height)
	}
	icccm.WmNormalHintsSet(w.xu, w.win.Id, hints)
}

func (w *OSWindow) Show() {
	if w.closed {
		return
	}
	w.win.Map()
}

func (w *OSWindow) Screen() (im gi.WinImage) {
	if w.closed {
		return
	}
	im = &Image{w.buffer}
	return
}

func (w *OSWindow) FlushImage(bounds ...image.Rectangle) {

	if w.closed {
		return
	}
	if w.buffer.Pixmap == 0 {
		w.bufferLck.Lock()
		if err := w.buffer.XSurfaceSet(w.win.Id); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		w.bufferLck.Unlock()
	}
	if len(bounds) > 0 {
		w.buffer.XPaintRects(w.win.Id, bounds...)
	} else {
		w.buffer.XDraw()
		w.buffer.XPaint(w.win.Id)
	}
}

func (w *OSWindow) Close() (err error) {
	if w.closed {
		return
	}
	w.win.Destroy()
	w.closed = true
	return
}

type WinImage struct {
	*xgraphics.Image
}

func (buffer WinImage) CopyRGBA(src *image.RGBA, r image.Rectangle) {
	// clip r against each image's bounds and move sp accordingly (see draw.clip())
	sp := src.Bounds().Min
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
