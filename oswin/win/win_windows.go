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

package win

import (
	"errors"
	"github.com/AllenDang/w32"
	"github.com/rcoreilly/goki/gi"
	"image"
	"runtime"
	"unsafe"
)

func init() {
	gi.FontLibrary.AddFontPaths("C:/Windows/Font")
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

const (
	WIN_CLASSNAME = "gi_win"
)

type OSWindow struct {
	EventData

	hwnd       w32.HWND
	buffer     *DIB
	bufferback *DIB
	events     chan interface{}

	keyDown   string // the most recently depressed key
	keysDown  map[string]bool
	keysStale bool // if true, keysDown may not reflect reality

	cursor gi.Cursor // most recently set cursor

	uiTasks chan func()
}

/*
go func(ready chan struct{}) {
		w, err = win.NewOSWindow(width, height)
		ready <- struct{}{}
		if winw, ok := w.(*win.OSWindow); ok {
			winw.HandleWndMessages()
		} else {
			panic("windows wgen returned non windows window")
		}
	}(ready)
	<-ready
*/

func makeTheWindow(width, height int) (w *OSWindow, err error) {

	err = RegClassOnlyOnce(WIN_CLASSNAME)
	if err != nil {
		return
	}

	cr := &w32.RECT{
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT + int32(width),
		w32.CW_USEDEFAULT + int32(height),
	}
	w32.AdjustWindowRectEx(cr, w32.WS_OVERLAPPEDWINDOW, false, w32.WS_EX_CLIENTEDGE)
	width = int(cr.Right - cr.Left)
	height = int(cr.Bottom - cr.Top)
	hwnd, err := CreateWindow(WIN_CLASSNAME, nil, w32.WS_EX_CLIENTEDGE, w32.WS_OVERLAPPEDWINDOW, width, height)
	if err != nil {
		return
	}

	w = &OSWindow{
		hwnd:       hwnd,
		buffer:     NewDIB(image.Rect(0, 0, width, height)),
		bufferback: NewDIB(image.Rect(0, 0, width, height)),
		events:     make(chan interface{}, 16),
		keysDown:   make(map[string]bool),
		uiTasks:    make(chan func(), 8),
	}
	w.InitEventData()

	RegMsgHandler(w)

	w.Center()

	return
}

func NewOSWindow(width, height int) (w *OSWindow, err error) {
	ready := make(chan error, 1)

	go func(ready chan error) {
		runtime.LockOSThread()
		var err error
		w, err = makeTheWindow(width, height)
		ready <- err
		if err != nil {
			return
		}
		w.HandleWndMessages()
	}(ready)

	err = <-ready
	return
}

func (this *OSWindow) SetTitle(title string) {
	w32.SetWindowText(this.hwnd, title)
}

func (this *OSWindow) SetSize(width, height int) {
	x, y := this.Pos()
	cr := &w32.RECT{0, 0, int32(width), int32(height)}
	w32.AdjustWindowRectEx(cr, w32.WS_OVERLAPPEDWINDOW, false, w32.WS_EX_CLIENTEDGE)
	width = int(cr.Right - cr.Left)
	height = int(cr.Bottom - cr.Top)
	w32.MoveWindow(this.hwnd, x, y, width, height, true)
}

func (this *OSWindow) Size() (width, height int) {
	bounds := this.buffer.Bounds()
	return bounds.Dx(), bounds.Dy()
}

func (w *OSWindow) LockSize(lock bool) {
	prevStyle := int(w32.GetWindowLongPtr(w.hwnd, w32.GWL_STYLE))
	if lock {
		prevStyle &= ^(w32.WS_MAXIMIZEBOX | w32.WS_SIZEBOX)
	} else {
		prevStyle |= w32.WS_MAXIMIZEBOX | w32.WS_SIZEBOX
	}
	w32.SetWindowLongPtr(w.hwnd, w32.GWL_STYLE, uintptr(prevStyle))
}

func (this *OSWindow) Show() {
	w32.ShowWindow(this.hwnd, w32.SW_SHOWDEFAULT)
}

func (this *OSWindow) Screen() gi.WinImage {
	return this.buffer
}

func (this *OSWindow) FlushImage(bounds ...image.Rectangle) {
	if this.buffer.Bounds().Empty() {
		return // happens when window is minimised
	}
	this.bufferback = NewDIB(this.buffer.Bounds())
	*this.bufferback = *this.buffer

	hdc := w32.GetDC(this.hwnd)
	this.blitImage(hdc, this.buffer)
	w32.DeleteDC(hdc)
}

func (this *OSWindow) EventChan() <-chan interface{} {
	return this.events
}

func (this *OSWindow) Close() error {
	UnRegMsgHandler(this.hwnd)
	err := w32.DestroyWindow(this.hwnd)
	if err == false {
		return errors.New("Error closing window")
	}
	return nil
}

/////////////////////////////
// Non - interface methods
/////////////////////////////

func (this *OSWindow) blitImage(hdc w32.HDC, buffer *DIB) {
	bounds := buffer.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	var bi w32.BITMAPINFO
	bi.BmiHeader.BiSize = uint32(unsafe.Sizeof(bi.BmiHeader))
	bi.BmiHeader.BiWidth = int32(width)
	bi.BmiHeader.BiHeight = int32(-height)
	bi.BmiHeader.BiPlanes = 1
	bi.BmiHeader.BiBitCount = 32
	bi.BmiHeader.BiCompression = w32.BI_RGB

	w32.SetDIBitsToDevice(hdc,
		0, 0,
		width, height,
		0, 0,
		0, uint(height),
		buffer.Pix, &bi,
		w32.DIB_RGB_COLORS,
	)
}

func (this *OSWindow) HandleWndMessages() {
	var m w32.MSG

	for w32.GetMessage(&m, this.hwnd, 0, 0) != 0 {
		w32.TranslateMessage(&m)
		w32.DispatchMessage(&m)
	}
}

func (this *OSWindow) Pos() (x, y int) {
	rect := w32.GetWindowRect(this.hwnd)
	return int(rect.Left), int(rect.Top)
}

func (this *OSWindow) SetPos(x, y int) {
	w, h := this.Size()
	if w == 0 {
		w = 100
	}
	if h == 0 {
		h = 25
	}
	w32.MoveWindow(this.hwnd, x, y, w, h, true)
}

func (this *OSWindow) Center() {
	sWidth := w32.GetSystemMetrics(w32.SM_CXFULLSCREEN)
	sHeight := w32.GetSystemMetrics(w32.SM_CYFULLSCREEN)

	if sWidth != 0 && sHeight != 0 {
		w, h := this.Size()
		this.SetPos((sWidth/2)-(w/2), (sHeight/2)-(h/2))
	}
}

func (this *OSWindow) Repaint() {
	hdc := w32.GetDC(this.hwnd)
	this.blitImage(hdc, this.bufferback)
	w32.DeleteDC(hdc)
}

func (this *OSWindow) onUiThread(f func()) {
	this.uiTasks <- f
	w32.PostMessage(this.hwnd, WDEM_UI_THREAD, uintptr(0), uintptr(0))
}
