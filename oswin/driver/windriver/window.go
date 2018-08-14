// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

// TODO: implement a back buffer.

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"syscall"
	"unsafe"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/driver/internal/drawer"
	"github.com/goki/gi/oswin/driver/internal/event"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
	"golang.org/x/image/math/f64"
)

type windowImpl struct {
	oswin.WindowBase
	hwnd syscall.Handle

	event.Deque

	closeReqFunc   func(win oswin.Window)
	closeCleanFunc func(win oswin.Window)
}

// for sending any kind of event
func sendEvent(hwnd syscall.Handle, ev oswin.Event) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	if w == nil {
		return
	}
	ev.Init()
	w.Send(ev)
}

// for sending window.Event's
func sendWindowEvent(w *windowImpl, act window.Actions) {
	winEv := window.Event{
		Action: act,
	}
	winEv.Init()
	w.Send(&winEv)
}

func (w *windowImpl) Upload(dp image.Point, src oswin.Image, sr image.Rectangle) {
	src.(*imageImpl).preUpload()
	defer src.(*imageImpl).postUpload()

	w.execCmd(&cmd{
		id:    cmdUpload,
		dp:    dp,
		image: src.(*imageImpl),
		sr:    sr,
	})
}

func (w *windowImpl) Fill(dr image.Rectangle, src color.Color, op draw.Op) {
	w.execCmd(&cmd{
		id:    cmdFill,
		dr:    dr,
		color: src,
		op:    op,
	})
}

func (w *windowImpl) Draw(src2dst f64.Aff3, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if op != draw.Src && op != draw.Over {
		// TODO:
		return
	}
	w.execCmd(&cmd{
		id:      cmdDraw,
		src2dst: src2dst,
		texture: src.(*textureImpl).bitmap,
		sr:      sr,
		op:      op,
	})
}

func (w *windowImpl) DrawUniform(src2dst f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if op != draw.Src && op != draw.Over {
		// TODO:
		return
	}
	w.execCmd(&cmd{
		id:      cmdDrawUniform,
		src2dst: src2dst,
		color:   src,
		sr:      sr,
		op:      op,
	})
}

func drawWindow(dc syscall.Handle, src2dst f64.Aff3, src interface{}, sr image.Rectangle, op draw.Op) (retErr error) {
	var dr image.Rectangle
	if src2dst[1] != 0 || src2dst[3] != 0 {
		// general drawing
		dr = sr.Sub(sr.Min)

		prevmode, err := _SetGraphicsMode(dc, _GM_ADVANCED)
		if err != nil {
			return err
		}
		defer func() {
			_, err := _SetGraphicsMode(dc, prevmode)
			if retErr == nil {
				retErr = err
			}
		}()

		x := _XFORM{
			eM11: +float32(src2dst[0]),
			eM12: -float32(src2dst[1]),
			eM21: -float32(src2dst[3]),
			eM22: +float32(src2dst[4]),
			eDx:  +float32(src2dst[2]),
			eDy:  +float32(src2dst[5]),
		}
		err = _SetWorldTransform(dc, &x)
		if err != nil {
			return err
		}
		defer func() {
			err := _ModifyWorldTransform(dc, nil, _MWT_IDENTITY)
			if retErr == nil {
				retErr = err
			}
		}()
	} else if src2dst[0] == 1 && src2dst[4] == 1 {
		// copy bitmap
		dr = sr.Add(image.Point{int(src2dst[2]), int(src2dst[5])})
	} else {
		// scale bitmap
		dstXMin := float64(sr.Min.X)*src2dst[0] + src2dst[2]
		dstXMax := float64(sr.Max.X)*src2dst[0] + src2dst[2]
		if dstXMin > dstXMax {
			// TODO: check if this (and below) works when src2dst[0] < 0.
			dstXMin, dstXMax = dstXMax, dstXMin
		}
		dstYMin := float64(sr.Min.Y)*src2dst[4] + src2dst[5]
		dstYMax := float64(sr.Max.Y)*src2dst[4] + src2dst[5]
		if dstYMin > dstYMax {
			// TODO: check if this (and below) works when src2dst[4] < 0.
			dstYMin, dstYMax = dstYMax, dstYMin
		}
		dr = image.Rectangle{
			image.Point{int(math.Floor(dstXMin)), int(math.Floor(dstYMin))},
			image.Point{int(math.Ceil(dstXMax)), int(math.Ceil(dstYMax))},
		}
	}
	switch s := src.(type) {
	case syscall.Handle:
		return copyBitmapToDC(dc, dr, s, sr, op)
	case color.Color:
		return fill(dc, dr, s, op)
	}
	return fmt.Errorf("unsupported type %T", src)
}

func (w *windowImpl) Copy(dp image.Point, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Copy(w, dp, src, sr, op, opts)
}

func (w *windowImpl) Scale(dr image.Rectangle, src oswin.Texture, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	drawer.Scale(w, dr, src, sr, op, opts)
}

func (w *windowImpl) Publish() oswin.PublishResult {
	// TODO
	return oswin.PublishResult{}
}

func (w *windowImpl) SetSize(sz image.Point) {
	ResizeClientRect(w.hwnd, sz)
}

func (w *windowImpl) SetPos(pos image.Point) {
	MoveWindowPos(w.hwnd, pos)
}

func (w *windowImpl) MainMenu() oswin.MainMenu {
	return nil
}

func (w *windowImpl) Raise() {
	_BringWindowToTop(w.hwnd)
}

func (w *windowImpl) Iconify() {
	_AnimateWindow(w.hwnd, 200, _AW_HIDE)
}

func (w *windowImpl) SetCloseReqFunc(fun func(win oswin.Window)) {
	w.closeReqFunc = fun
}

func (w *windowImpl) SetCloseCleanFunc(fun func(win oswin.Window)) {
	w.closeCleanFunc = fun
}

func (w *windowImpl) CloseReq() {
	if w.closeReqFunc != nil {
		w.closeReqFunc(w)
	} else {
		w.Close()
	}
}

func (w *windowImpl) CloseClean() {
	if w.closeCleanFunc != nil {
		w.closeCleanFunc(w)
	}
}

func (w *windowImpl) Close() {
	w.CloseClean()
	DeleteWindow(w.hwnd)
	theApp.DeleteWin(w.hwnd)
}

// cmd is used to carry parameters between user code
// and Windows message pump thread.
type cmd struct {
	id  int
	err error

	src2dst f64.Aff3
	sr      image.Rectangle
	dp      image.Point
	dr      image.Rectangle
	color   color.Color
	op      draw.Op
	texture syscall.Handle
	image   *imageImpl
}

const (
	cmdDraw = iota
	cmdFill
	cmdUpload
	cmdDrawUniform
)

var msgCmd = AddWindowMsg(handleCmd)

func (w *windowImpl) execCmd(c *cmd) {
	SendMessage(w.hwnd, msgCmd, 0, uintptr(unsafe.Pointer(c)))
	if c.err != nil {
		panic(fmt.Sprintf("execCmd faild for cmd.id=%d: %v", c.id, c.err)) // TODO handle errors
	}
}

func handleCmd(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) {
	c := (*cmd)(unsafe.Pointer(lParam))

	dc, err := _GetDC(hwnd)
	if err != nil {
		c.err = err
		return
	}
	defer _ReleaseDC(hwnd, dc)

	switch c.id {
	case cmdDraw:
		c.err = drawWindow(dc, c.src2dst, c.texture, c.sr, c.op)
	case cmdDrawUniform:
		c.err = drawWindow(dc, c.src2dst, c.color, c.sr, c.op)
	case cmdFill:
		c.err = fill(dc, c.dr, c.color, c.op)
	case cmdUpload:
		// TODO: adjust if dp is outside dst bounds, or sr is outside image bounds.
		dr := c.sr.Add(c.dp.Sub(c.sr.Min))
		c.err = copyBitmapToDC(dc, dr, c.image.hbitmap, c.sr, draw.Src)
	default:
		c.err = fmt.Errorf("unknown command id=%d", c.id)
	}
	return
}

var windowMsgs = map[uint32]func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr){
	_WM_SETFOCUS:         sendFocus,
	_WM_KILLFOCUS:        sendFocus,
	_WM_PAINT:            sendPaint,
	msgShow:              sendShow,
	_WM_WINDOWPOSCHANGED: sendSizeEvent,
	_WM_CLOSE:            sendCloseReq,
	_WM_DESTROY:          sendClose,
	_WM_QUIT:             sendQuit,

	_WM_LBUTTONDOWN: sendMouseEvent,
	_WM_LBUTTONUP:   sendMouseEvent,
	_WM_MBUTTONDOWN: sendMouseEvent,
	_WM_MBUTTONUP:   sendMouseEvent,
	_WM_RBUTTONDOWN: sendMouseEvent,
	_WM_RBUTTONUP:   sendMouseEvent,
	_WM_MOUSEMOVE:   sendMouseEvent,
	_WM_MOUSEWHEEL:  sendMouseEvent,

	_WM_KEYDOWN: sendKeyEvent,
	_WM_KEYUP:   sendKeyEvent,
	// TODO case _WM_SYSKEYDOWN, _WM_SYSKEYUP:
}

func AddWindowMsg(fn func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr)) uint32 {
	uMsg := currentUserWM.next()
	windowMsgs[uMsg] = func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
		fn(hwnd, uMsg, wParam, lParam)
		return 0
	}
	return uMsg
}

func windowWndProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	fn := windowMsgs[uMsg]
	if fn != nil {
		return fn(hwnd, uMsg, wParam, lParam)
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

type newWindowParams struct {
	opts *oswin.NewWindowOptions
	w    syscall.Handle
	err  error
}

func NewWindow(opts *oswin.NewWindowOptions) (syscall.Handle, error) {
	var p newWindowParams
	p.opts = opts
	SendAppMessage(msgCreateWindow, 0, uintptr(unsafe.Pointer(&p)))
	return p.w, p.err
}

func DeleteWindow(hwnd syscall.Handle) {
	SendAppMessage(msgDeleteWindow, 0, uintptr(unsafe.Pointer(hwnd)))
}

const windowClass = "GoGi_Window"

func initWindowClass() (err error) {
	wcname, err := syscall.UTF16PtrFromString(windowClass)
	if err != nil {
		return err
	}
	_, err = _RegisterClass(&_WNDCLASS{
		LpszClassName: wcname,
		LpfnWndProc:   syscall.NewCallback(windowWndProc),
		HIcon:         hDefaultIcon,
		HCursor:       hDefaultCursor,
		HInstance:     hThisInstance,
		// TODO(andlabs): change this to something else? NULL? the hollow brush?
		HbrBackground: syscall.Handle(_COLOR_BTNFACE + 1),
	})
	return err
}

func newWindow(opts *oswin.NewWindowOptions) (syscall.Handle, error) {
	// TODO(brainman): convert windowClass to *uint16 once (in initWindowClass)
	wcname, err := syscall.UTF16PtrFromString(windowClass)
	if err != nil {
		return 0, err
	}
	title, err := syscall.UTF16PtrFromString(opts.GetTitle())
	if err != nil {
		return 0, err
	}
	hwnd, err := _CreateWindowEx(0,
		wcname, title,
		_WS_OVERLAPPEDWINDOW,
		int32(opts.Pos.X), int32(opts.Pos.Y),
		int32(opts.Size.X), int32(opts.Size.Y),
		0, 0, hThisInstance, 0)
	if err != nil {
		return 0, err
	}
	// TODO(andlabs): use proper nCmdShow
	// TODO(andlabs): call UpdateWindow()

	return hwnd, nil
}

// ResizeClientRect makes hwnd client rectangle given size
func ResizeClientRect(hwnd syscall.Handle, size image.Point) error {
	var cr, wr _RECT
	err := _GetClientRect(hwnd, &cr)
	if err != nil {
		return err
	}
	err = _GetWindowRect(hwnd, &wr)
	if err != nil {
		return err
	}
	w := (wr.Right - wr.Left) - (cr.Right - int32(size.X))
	h := (wr.Bottom - wr.Top) - (cr.Bottom - int32(size.Y))
	return _MoveWindow(hwnd, wr.Left, wr.Top, w, h, false)
}

// MoveWindowPos
func MoveWindowPos(hwnd syscall.Handle, pos image.Point) error {
	var wr _RECT
	err = _GetWindowRect(hwnd, &wr)
	if err != nil {
		return err
	}
	w := (wr.Right - wr.Left)
	h := (wr.Bottom - wr.Top)
	return _MoveWindow(hwnd, pos.X, pos.Y, w, h, false)
}

// Show shows a newly created window.  It makes the window appear on the
// screen, and sends an initial size event.
//
// This is a separate step from NewWindow to give the driver a chance
// to setup its internal state for a window before events start being
// delivered.
func Show(hwnd syscall.Handle) {
	SendMessage(hwnd, msgShow, 0, 0)
}

// this must be called in original app thread..
func deleteWindow(hwnd syscall.Handle) {
	_DestroyWindow(hwnd)
}

func sendFocus(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	switch uMsg {
	case _WM_SETFOCUS:
		bitflag.Clear(&w.Flag, int(oswin.Iconified))
		bitflag.Set(&w.Flag, int(oswin.Focus))
		sendWindowEvent(w, window.Focus)
	case _WM_KILLFOCUS:
		bitflag.Clear(&w.Flag, int(oswin.Focus))
		sendWindowEvent(w, window.DeFocus)
	default:
		panic(fmt.Sprintf("windriver: unexpected focus message: %d", uMsg))
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

func sendShow(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	_ShowWindow(hwnd, _SW_SHOWDEFAULT)
	sendSize(hwnd)
	return 0
}

func sendSizeEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	if _IsIconic(hwnd) {
		sendIconify(hwnd)
		return 0
	} else {
		wp := (*_WINDOWPOS)(unsafe.Pointer(lParam))
		if wp.Flags&_SWP_NOSIZE != 0 {
			return 0
		}
		sendSize(hwnd)
		return 0
	}
}

func sendIconify(hwnd syscall.Handle) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	bitflag.Set(&w.Flag, int(oswin.Iconified))
	bitflag.Clear(&w.Flag, int(oswin.Focus))
	sendWindowEvent(w, window.Iconify)
}

func sendSize(hwnd syscall.Handle) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()

	var r _RECT
	if err := _GetClientRect(hwnd, &r); err != nil {
		panic(err) // TODO(andlabs)
	}

	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)

	if width < 100 {
		width = 1000
	}
	if height < 100 {
		height = 1000
	}
	sz := image.Point{int(width), int(height)}
	ps := image.Point{int(r.Left), int(r.Top)}
	act := window.Resize // also resolved at higher level that has access to prev

	// todo: multiple screens
	sc := oswin.TheApp.Screen(0)
	ldpi := sc.LogicalDPI
	act := window.ActionsN

	if w.Sz != sz || w.LogDPI != ldpi {
		act = window.Resize
	} else if w.Pos != ps {
		act = window.Move
		// } else {
		// 	//		act = window.Resize // todo: for now safer to default to resize -- to catch the
		// filtering
	}

	w.Sz = sz
	w.Pos = ps
	w.PhysDPI = sc.PhysicalDPI
	w.LogDPI = ldpi
	w.Scrn = sc
	sendWindowEvent(w, act)
}

func sendCloseReq(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	go w.CloseReq()
	return 0 //
}

func sendClose(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	w.CloseClean()
	sendWindowEvent(w, window.Close)
	Release(hwnd)
	return 0
}

func sendPaint(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theApp.mu.Lock()
	w := theApp.windows[hwnd]
	theApp.mu.Unlock()
	bitflag.Clear(&w.Flag, int(oswin.Iconified))
	sendWindowEvent(hwnd, window.Paint)
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}
