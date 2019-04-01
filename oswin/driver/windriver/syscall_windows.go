// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"syscall"
	"unsafe"
)

type _COLORREF uint32

func _RGB(r, g, b byte) _COLORREF {
	return _COLORREF(r) | _COLORREF(g)<<8 | _COLORREF(b)<<16
}

type _POINT struct {
	X int32
	Y int32
}

type _RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type _MSG struct {
	HWND    syscall.Handle
	Message uint32
	Wparam  uintptr
	Lparam  uintptr
	Time    uint32
	Pt      _POINT
}

type _WNDCLASS struct {
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
}

type _WINDOWPOS struct {
	HWND            syscall.Handle
	HWNDInsertAfter syscall.Handle
	X               int32
	Y               int32
	Cx              int32
	Cy              int32
	Flags           uint32
}

type _BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type _RGBQUAD struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type _BITMAPINFO struct {
	Header _BITMAPINFOHEADER
	Colors [1]_RGBQUAD
}

type _BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

// ToUintptr helps to pass bf to syscall.Syscall.
func (bf _BLENDFUNCTION) ToUintptr() uintptr {
	return *((*uintptr)(unsafe.Pointer(&bf)))
}

type _XFORM struct {
	eM11 float32
	eM12 float32
	eM21 float32
	eM22 float32
	eDx  float32
	eDy  float32
}

type _DISPLAY_DEVICE struct {
	CB           uint32
	DeviceName   [32]byte
	DeviceString [128]byte
	StateFlags   uint32
	DeviceID     [128]byte
	DeviceKey    [128]byte
}

const (
	_WM_DESTROY          = 2
	_WM_SETFOCUS         = 7
	_WM_KILLFOCUS        = 8
	_WM_PAINT            = 15
	_WM_CLOSE            = 16
	_WM_QUIT             = 18
	_WM_SETCURSOR        = 32
	_WM_WINDOWPOSCHANGED = 71
	_WM_KEYDOWN          = 256
	_WM_KEYUP            = 257
	_WM_SYSKEYDOWN       = 260
	_WM_SYSKEYUP         = 261
	_WM_MOUSEMOVE        = 512
	_WM_MOUSEWHEEL       = 522
	_WM_LBUTTONDOWN      = 513
	_WM_LBUTTONUP        = 514
	_WM_RBUTTONDOWN      = 516
	_WM_RBUTTONUP        = 517
	_WM_MBUTTONDOWN      = 519
	_WM_MBUTTONUP        = 520
	_WM_USER             = 0x0400
)

const (
	_WS_OVERLAPPED       = 0x00000000
	_WS_CAPTION          = 0x00C00000
	_WS_SYSMENU          = 0x00080000
	_WS_THICKFRAME       = 0x00040000
	_WS_MINIMIZEBOX      = 0x00020000
	_WS_MAXIMIZEBOX      = 0x00010000
	_WS_OVERLAPPEDWINDOW = _WS_OVERLAPPED | _WS_CAPTION | _WS_SYSMENU | _WS_THICKFRAME | _WS_MINIMIZEBOX | _WS_MAXIMIZEBOX
)

const (
	_VK_SHIFT   = 16
	_VK_CONTROL = 17
	_VK_MENU    = 18
	_VK_LWIN    = 0x5B
	_VK_RWIN    = 0x5C
)

const (
	_MK_LBUTTON = 0x0001
	_MK_MBUTTON = 0x0010
	_MK_RBUTTON = 0x0002
)

const (
	_COLOR_BTNFACE = 15
)

const (
	_IDI_APPLICATION = 32512
)

const (
	_IDC_ARROW    = 32512
	_IDC_CROSS    = 32515
	_IDC_HAND     = 32649
	_IDC_HELP     = 32651
	_IDC_IBEAM    = 32513
	_IDC_NO       = 32648
	_IDC_SIZEALL  = 32646
	_IDC_SIZENESW = 32643
	_IDC_SIZENS   = 32645
	_IDC_SIZENWSE = 32642
	_IDC_SIZEWE   = 32644
	_IDC_UPARROW  = 32516
	_IDC_WAIT     = 32514
)

const (
	_CW_USEDEFAULT = 0x80000000 - 0x100000000

	_AW_HIDE = 0x00010000

	_HWND_MESSAGE = syscall.Handle(^uintptr(2)) // -3

	_SWP_NOSIZE = 0x0001
)

const (
	_SW_HIDE        = 0
	_SW_MAXIMIZE    = 3
	_SW_SHOW        = 5
	_SW_MINIMIZE    = 6
	_SW_RESTORE     = 9
	_SW_SHOWDEFAULT = 10
)

const (
	_DISPLAY_DEVICE_ACTIVE         = 1
	_DISPLAY_DEVICE_PRIMARY_DEVICE = 4
	_HORZSIZE                      = 4
	_VERTSIZE                      = 6
	_HORZRES                       = 8
	_VERTRES                       = 10
	_LOGPIXELSX                    = 88
	_LOGPIXELSY                    = 90
)

const (
	_BI_RGB         = 0
	_DIB_RGB_COLORS = 0

	_AC_SRC_OVER  = 0x00
	_AC_SRC_ALPHA = 0x01

	_SRCCOPY = 0x00cc0020

	_SHADEBLENDCAPS = 120
	_SB_NONE        = 0
	_WHEEL_DELTA    = 120
)

const (
	_GM_COMPATIBLE = 1
	_GM_ADVANCED   = 2

	_MWT_IDENTITY = 1

	_PROCESS_PER_MONITOR_DPI_AWARE = 2
)

const (
	_CF_UNICODETEXT = 13
)

const (
	_GMEM_MOVEABLE = 0x0002
)

func _GET_X_LPARAM(lp uintptr) int32 {
	return int32(_LOWORD(lp))
}

func _GET_Y_LPARAM(lp uintptr) int32 {
	return int32(_HIWORD(lp))
}

func _GET_WHEEL_DELTA_WPARAM(lp uintptr) int16 {
	return int16(_HIWORD(lp))
}

func _LOWORD(l uintptr) uint16 {
	return uint16(uint32(l))
}

func _HIWORD(l uintptr) uint16 {
	return uint16(uint32(l >> 16))
}

// notes to self
// UINT = uint32
// callbacks = uintptr
// strings = *uint16

//sys	_GetDC(hwnd syscall.Handle) (dc syscall.Handle, err error) = user32.GetDC
//sys	_ReleaseDC(hwnd syscall.Handle, dc syscall.Handle) (err error) = user32.ReleaseDC
//sys	_sendMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) = user32.SendMessageW

//sys	_CreateWindowEx(exstyle uint32, className *uint16, windowText *uint16, style uint32, x int32, y int32, width int32, height int32, parent syscall.Handle, menu syscall.Handle, hInstance syscall.Handle, lpParam uintptr) (hwnd syscall.Handle, err error) = user32.CreateWindowExW
//sys	_DefWindowProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) = user32.DefWindowProcW
//sys	_DestroyWindow(hwnd syscall.Handle) (err error) = user32.DestroyWindow
//sys	_DispatchMessage(msg *_MSG) (ret int32) = user32.DispatchMessageW
//sys	_GetClientRect(hwnd syscall.Handle, rect *_RECT) (err error) = user32.GetClientRect
//sys	_GetWindowRect(hwnd syscall.Handle, rect *_RECT) (err error) = user32.GetWindowRect
//sys	_SetWindowText(hwnd syscall.Handle, windowText *uint16) (ok bool) = user32.SetWindowTextW
//sys   _GetKeyboardLayout(threadID uint32) (locale syscall.Handle) = user32.GetKeyboardLayout
//sys   _GetKeyboardState(lpKeyState *byte) (err error) = user32.GetKeyboardState
//sys	_GetKeyState(virtkey int32) (keystatus int16) = user32.GetKeyState
//sys	_GetMessage(msg *_MSG, hwnd syscall.Handle, msgfiltermin uint32, msgfiltermax uint32) (ret int32, err error) [failretval==-1] = user32.GetMessageW
//sys	_LoadCursor(hInstance syscall.Handle, cursorName uintptr) (cursor syscall.Handle) = user32.LoadCursorA
//sys	_LoadIcon(hInstance syscall.Handle, iconName uintptr) (icon syscall.Handle, err error) = user32.LoadIconW
//sys	_SetCursor(hinst syscall.Handle) (prevcurs syscall.Handle) = user32.SetCursor
//sys	_ShowCursor(show bool) (dispcnt int) = user32.ShowCursor
//sys	_MoveWindow(hwnd syscall.Handle, x int32, y int32, w int32, h int32, repaint bool) (err error) = user32.MoveWindow
//sys	_BringWindowToTop(hwnd syscall.Handle) (err error) = user32.BringWindowToTop
//sys	_PostMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult bool) = user32.PostMessageW
//sys   _PostQuitMessage(exitCode int32) = user32.PostQuitMessage
//sys	_RegisterClass(wc *_WNDCLASS) (atom uint16, err error) = user32.RegisterClassW
//sys	_ShowWindow(hwnd syscall.Handle, cmdshow int32) (wasvisible bool) = user32.ShowWindow
//sys	_SetActiveWindow(hwnd syscall.Handle) (prev syscall.Handle) = user32.SetActiveWindow
//sys	_SetFocus(hwnd syscall.Handle) (prev syscall.Handle) = user32.SetFocus
//sys	_SetForegroundWindow(hwnd syscall.Handle) (ok bool) = user32.SetForegroundWindow
//sys	_IsIconic(hwnd syscall.Handle) (iconic bool) = user32.IsIconic
//sys	_IsWindowVisible(hwnd syscall.Handle) (vis bool) = user32.IsWindowVisible
//sys	_ScreenToClient(hwnd syscall.Handle, lpPoint *_POINT) (ok bool) = user32.ScreenToClient
//sys   _ToUnicodeEx(wVirtKey uint32, wScanCode uint32, lpKeyState *byte, pwszBuff *uint16, cchBuff int32, wFlags uint32, dwhkl syscall.Handle) (ret int32) = user32.ToUnicodeEx
//sys	_TranslateMessage(msg *_MSG) (done bool) = user32.TranslateMessage
//sys	_OpenClipboard(hwnd syscall.Handle) (opened bool) = user32.OpenClipboard
//sys	_CloseClipboard() (closed bool) = user32.CloseClipboard
//sys	_EmptyClipboard() (empty bool) = user32.EmptyClipboard
//sys	_SetClipboardData(uFormat uint32, hMem syscall.Handle) (hRes syscall.Handle) = user32.SetClipboardData
//sys	_GetClipboardData(uFormat uint32) (hMem syscall.Handle) = user32.GetClipboardData
//sys	_IsClipboardFormatAvailable(uFormat uint32) (avail bool) = user32.IsClipboardFormatAvailable
//sys	_GlobalLock(hMem syscall.Handle) (data *uint16) = kernel32.GlobalLock
//sys	_GlobalUnlock(hMem syscall.Handle) (unlocked bool) = kernel32.GlobalUnlock
//sys	_GlobalAlloc(uFlags uint32, size uintptr) (hMem syscall.Handle) = kernel32.GlobalAlloc
//sys	_GlobalFree(hMem syscall.Handle) = kernel32.GlobalFree
//sys	_CopyMemory(dest uintptr, src uintptr, sz uintptr) = kernel32.RtlCopyMemory

//sys	_AlphaBlend(dcdest syscall.Handle, xoriginDest int32, yoriginDest int32, wDest int32, hDest int32, dcsrc syscall.Handle, xoriginSrc int32, yoriginSrc int32, wsrc int32, hsrc int32, ftn uintptr) (err error) = msimg32.AlphaBlend
//sys	_BitBlt(dcdest syscall.Handle, xdest int32, ydest int32, width int32, height int32, dcsrc syscall.Handle, xsrc int32, ysrc int32, rop uint32) (err error) = gdi32.BitBlt
//sys	_CreateCompatibleBitmap(dc syscall.Handle, width int32, height int32) (bitmap syscall.Handle, err error) = gdi32.CreateCompatibleBitmap
//sys	_CreateCompatibleDC(dc syscall.Handle) (newdc syscall.Handle, err error) = gdi32.CreateCompatibleDC
//sys	_CreateDIBSection(dc syscall.Handle, bmi *_BITMAPINFO, usage uint32, bits **byte, section syscall.Handle, offset uint32) (bitmap syscall.Handle, err error) = gdi32.CreateDIBSection
//sys	_CreateSolidBrush(color _COLORREF) (brush syscall.Handle, err error) = gdi32.CreateSolidBrush
//sys	_DeleteDC(dc syscall.Handle) (err error) = gdi32.DeleteDC
//sys	_DeleteObject(object syscall.Handle) (err error) = gdi32.DeleteObject
//sys	_FillRect(dc syscall.Handle, rc *_RECT, brush syscall.Handle) (err error) = user32.FillRect
//sys	_ModifyWorldTransform(dc syscall.Handle, x *_XFORM, mode uint32) (err error) = gdi32.ModifyWorldTransform
//sys	_SelectObject(dc syscall.Handle, gdiobj syscall.Handle) (newobj syscall.Handle, err error) = gdi32.SelectObject
//sys	_SetGraphicsMode(dc syscall.Handle, mode int32) (oldmode int32, err error) = gdi32.SetGraphicsMode
//sys	_SetWorldTransform(dc syscall.Handle, x *_XFORM) (err error) = gdi32.SetWorldTransform
//sys	_StretchBlt(dcdest syscall.Handle, xdest int32, ydest int32, wdest int32, hdest int32, dcsrc syscall.Handle, xsrc int32, ysrc int32, wsrc int32, hsrc int32, rop uint32) (err error) = gdi32.StretchBlt
//sys	_GetDeviceCaps(dc syscall.Handle, index int32) (ret int32) = gdi32.GetDeviceCaps
//sys	_SetProcessDpiAwareness(pdpi uint32) (ret int32) = shcore.SetProcessDpiAwareness
//sys	_GetDpiForWindow(hwnd syscall.Handle) (ret uint32) = user32.GetDpiForWindow
//sys	_EnumDisplayDevices(lpdevice uintptr, idevnum uint32, dispdev *_DISPLAY_DEVICE, dwflags uint32) (ok bool) = user32.EnumDisplayDevicesA
