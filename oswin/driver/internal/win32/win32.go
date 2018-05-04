// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

// Package win32 implements a partial oswin app driver using the Win32 API.
// It provides window, lifecycle, key, and mouse management, but no drawing.
// That is left to windriver (using GDI) or gldriver (using DirectX via ANGLE).
package win32

import (
	"fmt"
	"image"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/goki/goki/gi/oswin"
	"github.com/goki/goki/gi/oswin/key"
	"github.com/goki/goki/gi/oswin/lifecycle"
	"github.com/goki/goki/gi/oswin/mouse"
	"github.com/goki/goki/gi/oswin/paint"
	"github.com/goki/goki/gi/oswin/window"
	"golang.org/x/mobile/geom"
)

// appWND is the handle to the "AppWindow".  The window encapsulates all
// oswin.Window operations in an actual Windows window so they all run on the
// main thread.  Since any messages sent to a window will be executed on the
// main thread, we can safely use the messages below.
var appHWND syscall.Handle

const (
	msgCreateWindow = _WM_USER + iota
	msgMainCallback
	msgShow
	msgQuit
	msgLast
)

// userWM is used to generate private (WM_USER and above) window message IDs
// for use by appWindowWndProc and windowWndProc.
type userWM struct {
	sync.Mutex
	id uint32
}

func (m *userWM) next() uint32 {
	m.Lock()
	if m.id == 0 {
		m.id = msgLast
	}
	r := m.id
	m.id++
	m.Unlock()
	return r
}

var currentUserWM userWM

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

// ResizeClientRect makes hwnd client rectangle opts.Width by opts.Height in size.
func ResizeClientRect(hwnd syscall.Handle, opts *oswin.NewWindowOptions) error {
	var cr, wr _RECT
	err := _GetClientRect(hwnd, &cr)
	if err != nil {
		return err
	}
	err = _GetWindowRect(hwnd, &wr)
	if err != nil {
		return err
	}
	w := (wr.Right - wr.Left) - (cr.Right - int32(opts.Size.X))
	h := (wr.Bottom - wr.Top) - (cr.Bottom - int32(opts.Size.Y))
	return _MoveWindow(hwnd, wr.Left, wr.Top, w, h, false)
}

// Show shows a newly created window.
// It sends the appropriate lifecycle events, makes the window appear
// on the screen, and sends an initial size event.
//
// This is a separate step from NewWindow to give the driver a chance
// to setup its internal state for a window before events start being
// delivered.
func Show(hwnd syscall.Handle) {
	SendMessage(hwnd, msgShow, 0, 0)
}

func Release(hwnd syscall.Handle) {
	// TODO(andlabs): check for errors from this?
	// TODO(andlabs): remove unsafe
	_DestroyWindow(hwnd)
	// TODO(andlabs): what happens if we're still painting?
}

func sendFocus(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	switch uMsg {
	case _WM_SETFOCUS:
		LifecycleEvent(hwnd, lifecycle.StageFocused)
	case _WM_KILLFOCUS:
		LifecycleEvent(hwnd, lifecycle.StageVisible)
	default:
		panic(fmt.Sprintf("unexpected focus message: %d", uMsg))
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

func sendShow(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	LifecycleEvent(hwnd, lifecycle.StageVisible)
	_ShowWindow(hwnd, _SW_SHOWDEFAULT)
	sendSize(hwnd)
	return 0
}

func sendSizeEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	wp := (*_WINDOWPOS)(unsafe.Pointer(lParam))
	if wp.Flags&_SWP_NOSIZE != 0 {
		return 0
	}
	sendSize(hwnd)
	return 0
}

func sendSize(hwnd syscall.Handle) {
	var r _RECT
	if err := _GetClientRect(hwnd, &r); err != nil {
		panic(err) // TODO(andlabs)
	}

	width := int(r.Right - r.Left)
	height := int(r.Bottom - r.Top)

	// TODO(andlabs): don't assume that PixelsPerPt == 1
	SizeEvent(hwnd, window.Event{
		WidthPx:     width,
		HeightPx:    height,
		WidthPt:     geom.Pt(width),
		HeightPt:    geom.Pt(height),
		PixelsPerPt: 1,
	})
}

func sendClose(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	LifecycleEvent(hwnd, lifecycle.StageDead)
	return 0
}

var lastMouseClickEvent oswin.Event
var lastMouseEvent oswin.Event

func sendMouseEvent(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {

	where := image.Point{int(_GET_X_LPARAM(lParam)), int(_GET_Y_LPARAM(lParam))}
	mods := keyModifiers()

	button := mouse.NoButton
	switch uMsg {
	case _WM_MOUSEMOVE:
		// No-op.
	case _WM_LBUTTONDOWN, _WM_LBUTTONUP:
		button = mouse.Left
	case _WM_MBUTTONDOWN, _WM_MBUTTONUP:
		button = mouse.Middle
	case _WM_RBUTTONDOWN, _WM_RBUTTONUP:
		button = mouse.Right
	}

	var event oswin.Event
	switch uMsg {
	case _WM_MOUSEMOVE:
		// todo: drag!
		event = &mouse.MoveEvent{
			Event: mouse.Event{
				Where:     where,
				Button:    button,
				Action:    mouse.Move,
				Modifiers: mods,
			},
			From: from,
		}
	case _WM_LBUTTONDOWN, _WM_MBUTTONDOWN, _WM_RBUTTONDOWN:
		event = &mouse.Event{
			Where:     where,
			Button:    button,
			Action:    mouse.Press,
			Modifiers: mods,
		}
		event.SetTime()
		lastMouseClickEvent = event
	case _WM_LBUTTONUP, _WM_MBUTTONUP, _WM_RBUTTONUP:
		event = &mouse.Event{
			Where:     where,
			Button:    button,
			Action:    mouse.Release,
			Modifiers: mods,
		}
		event.SetTime()
		lastMouseClickEvent = event
	case _WM_MOUSEWHEEL:
		// TODO: handle horizontal scrolling
		delta := _GET_WHEEL_DELTA_WPARAM(wParam) / _WHEEL_DELTA
		// Convert from screen to window coordinates.
		p := _POINT{
			int32(where.X),
			int32(where.Y),
		}
		_ScreenToClient(hwnd, &p)
		where.X = float32(p.X)
		where.Y = float32(p.Y)

		event = &mouse.ScrollEvent{
			Event: mouse.Event{
				Where:     where,
				Button:    button,
				Action:    mouse.Scroll,
				Modifiers: mods,
			},
			Delta: image.Point{0, delta}, // only vert
		}
	default:
		panic("sendMouseEvent() called on non-mouse message")
	}

	MouseEvent(hwnd, event)

	return 0
}

// Precondition: this is called in immediate response to the message that triggered the event (so not after w.Send).
func keyModifiers() (m key.Modifiers) {
	down := func(x int32) bool {
		// GetKeyState gets the key state at the time of the message, so this is what we want.
		return _GetKeyState(x)&0x80 != 0
	}

	if down(_VK_CONTROL) {
		m |= 1 << key.Control
	}
	if down(_VK_MENU) {
		m |= 1 << key.Alt
	}
	if down(_VK_SHIFT) {
		m |= 1 << key.Shift
	}
	if down(_VK_LWIN) || down(_VK_RWIN) {
		m |= 1 << key.Meta
	}
	return m
}

var (
	MouseEvent     func(hwnd syscall.Handle, e *mouse.Event)
	PaintEvent     func(hwnd syscall.Handle, e *paint.Event)
	SizeEvent      func(hwnd syscall.Handle, e *window.Event)
	KeyEvent       func(hwnd syscall.Handle, e *key.Event)
	LifecycleEvent func(hwnd syscall.Handle, e *lifecycle.Stage)
)

func sendPaint(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	PaintEvent(hwnd, paint.Event{})
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

var screenMsgs = map[uint32]func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr){}

func AddScreenMsg(fn func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr)) uint32 {
	uMsg := currentUserWM.next()
	screenMsgs[uMsg] = func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
		fn(hwnd, uMsg, wParam, lParam)
		return 0
	}
	return uMsg
}

func appWindowWndProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	switch uMsg {
	case msgCreateWindow:
		p := (*newWindowParams)(unsafe.Pointer(lParam))
		p.w, p.err = newWindow(p.opts)
	case msgMainCallback:
		go func() {
			mainCallback()
			SendScreenMessage(msgQuit, 0, 0)
		}()
	case msgQuit:
		_PostQuitMessage(0)
	}
	fn := screenMsgs[uMsg]
	if fn != nil {
		return fn(hwnd, uMsg, wParam, lParam)
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

//go:uintptrescapes

func SendScreenMessage(uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	return SendMessage(appHWND, uMsg, wParam, lParam)
}

var windowMsgs = map[uint32]func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr){
	_WM_SETFOCUS:         sendFocus,
	_WM_KILLFOCUS:        sendFocus,
	_WM_PAINT:            sendPaint,
	msgShow:              sendShow,
	_WM_WINDOWPOSCHANGED: sendSizeEvent,
	_WM_CLOSE:            sendClose,

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
	SendScreenMessage(msgCreateWindow, 0, uintptr(unsafe.Pointer(&p)))
	return p.w, p.err
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

func initAppWindow() (err error) {
	const appWindowClass = "GoGi_AppWindow"
	swc, err := syscall.UTF16PtrFromString(appWindowClass)
	if err != nil {
		return err
	}
	emptyString, err := syscall.UTF16PtrFromString("")
	if err != nil {
		return err
	}
	wc := _WNDCLASS{
		LpszClassName: swc,
		LpfnWndProc:   syscall.NewCallback(appWindowWndProc),
		HIcon:         hDefaultIcon,
		HCursor:       hDefaultCursor,
		HInstance:     hThisInstance,
		HbrBackground: syscall.Handle(_COLOR_BTNFACE + 1),
	}
	_, err = _RegisterClass(&wc)
	if err != nil {
		return err
	}
	appHWND, err = _CreateWindowEx(0,
		swc, emptyString,
		_WS_OVERLAPPEDWINDOW,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		_CW_USEDEFAULT, _CW_USEDEFAULT,
		_HWND_MESSAGE, 0, hThisInstance, 0)
	if err != nil {
		return err
	}

	return nil
}

func ScreenSize() (width, height int) {
	width = 1024
	height = 768
	var wr _RECT
	err = _GetWindowRect(appHWND, &wr)
	if err != nil {
		width = int(wr.Right - wr.Left)
		height = int(wr.Bottom - wr.Top)
	}
	return
}

var (
	hDefaultIcon   syscall.Handle
	hDefaultCursor syscall.Handle
	hThisInstance  syscall.Handle
)

func initCommon() (err error) {
	hDefaultIcon, err = _LoadIcon(0, _IDI_APPLICATION)
	if err != nil {
		return err
	}
	hDefaultCursor, err = _LoadCursor(0, _IDC_ARROW)
	if err != nil {
		return err
	}
	// TODO(andlabs) hThisInstance
	return nil
}

//go:uintptrescapes

func SendMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	return sendMessage(hwnd, uMsg, wParam, lParam)
}

var mainCallback func()

func Main(f func()) (retErr error) {
	// It does not matter which OS thread we are on.
	// All that matters is that we confine all UI operations
	// to the thread that created the respective window.
	runtime.LockOSThread()

	if err := initCommon(); err != nil {
		return err
	}

	if err := initAppWindow(); err != nil {
		return err
	}
	defer func() {
		// TODO(andlabs): log an error if this fails?
		_DestroyWindow(appHWND)
		// TODO(andlabs): unregister window class
	}()

	if err := initWindowClass(); err != nil {
		return err
	}

	// Prime the pump.
	mainCallback = f
	_PostMessage(appHWND, msgMainCallback, 0, 0)

	// Main message pump.
	var m _MSG
	for {
		done, err := _GetMessage(&m, 0, 0, 0)
		if err != nil {
			return fmt.Errorf("win32 GetMessage failed: %v", err)
		}
		if done == 0 { // WM_QUIT
			break
		}
		_TranslateMessage(&m)
		_DispatchMessage(&m)
	}

	return nil
}
