// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
)

var theApp = &appImpl{
	windows: make(map[syscall.Handle]*windowImpl),
	winlist: make([]*windowImpl, 0),
	screens: make([]*oswin.Screen, 0),
	name:    "GoGi",
}

type appImpl struct {
	mu            sync.Mutex
	windows       map[syscall.Handle]*windowImpl
	winlist       []*windowImpl
	screens       []*oswin.Screen
	ctxtwin       *windowImpl
	name          string
	about         string
	quitting      bool          // set to true when quitting and closing windows
	quitCloseCnt  chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc   func()
	quitCleanFunc func()
}

func (*appImpl) NewImage(size image.Point) (oswin.Image, error) {
	// Image length must fit in BITMAPINFO.Header.SizeImage (uint32), as
	// well as in Go slice length (int). It's easiest to be consistent
	// between 32-bit and 64-bit, so we just use int32.
	const (
		maxInt32  = 0x7fffffff
		maxBufLen = maxInt32
	)
	if size.X < 0 || size.X > maxInt32 || size.Y < 0 || size.Y > maxInt32 || int64(size.X)*int64(size.Y)*4 > maxBufLen {
		return nil, fmt.Errorf("windriver: invalid image size %v", size)
	}

	hbitmap, bitvalues, err := mkbitmap(size)
	if err != nil {
		return nil, err
	}
	bufLen := 4 * size.X * size.Y
	array := (*[maxBufLen]byte)(unsafe.Pointer(bitvalues))
	buf := (*array)[:bufLen:bufLen]
	return &imageImpl{
		hbitmap: hbitmap,
		buf:     buf,
		rgba: image.RGBA{
			Pix:    buf,
			Stride: 4 * size.X,
			Rect:   image.Rectangle{Max: size},
		},
		size: size,
	}, nil
}

func (*appImpl) NewTexture(w oswin.Window, size image.Point) (oswin.Texture, error) {
	return newTexture(w, size)
}

func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	if opts == nil {
		opts = &oswin.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	// fmt.Printf("new window opts pos: %v  size: %v\n", opts.Pos, opts.Size)

	// todo: multiple screens..
	sc := app.Screen(0)
	dpi := sc.PhysicalDPI
	ldpi := sc.LogicalDPI

	w := &windowImpl{
		WindowBase: oswin.WindowBase{
			Pos:     opts.Pos,
			PhysDPI: dpi,
			LogDPI:  ldpi,
		},
	}

	var err error
	w.hwnd, err = NewWindow(opts)
	if err != nil {
		return nil, err
	}

	app.mu.Lock()
	app.windows[w.hwnd] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	err = ConfigWindowGeom(w.hwnd, opts.Pos, opts.Size)
	if err != nil {
		return nil, err
	}

	Show(w.hwnd)

	if procGetDpiForWindow.Find() == nil { // has it
		dpi = float32(_GetDpiForWindow(w.hwnd))
		if dpi != sc.PhysicalDPI {
			fmt.Printf("Window DPI: %v is different from screen: %v\n", dpi, sc.PhysicalDPI)
			sc.PhysicalDPI = dpi
			// todo: need to be able to trigger an update of ldpi from settings
		}
	}

	return w, nil
}

func (app *appImpl) DeleteWin(id syscall.Handle) {
	app.mu.Lock()
	defer app.mu.Unlock()
	win, ok := app.windows[id]
	if !ok {
		return
	}
	for i, w := range app.winlist {
		if w == win {
			app.winlist = append(app.winlist[:i], app.winlist[i+1:]...)
			break
		}
	}
	delete(app.windows, id)
}

func NullTermToString(b []byte) string {
	i := bytes.IndexByte(b, 0)
	if i == -1 {
		return ""
	}
	return string(b[0:i])
}

func (app *appImpl) initScreens() {
	// https://blogs.windows.com/buildingapps/2016/10/24/high-dpi-scaling-improvements-for-desktop-applications-and-mixed-mode-dpi-scaling-in-the-windows-10-anniversary-update/

	// https://github.com/Microsoft/Windows-classic-samples/tree/master/Samples/DPIAwarenessPerWindow/cpp
	// https://msdn.microsoft.com/en-us/library/windows/desktop/hh447398(v=vs.85).aspx

	if procSetProcessDpiAwareness.Find() == nil { // returns err -- has it
		_SetProcessDpiAwareness(_PROCESS_PER_MONITOR_DPI_AWARE)
	}

	ddinfo := _DISPLAY_DEVICE{}
	ddinfo.CB = uint32(unsafe.Sizeof(ddinfo))

	// todo: this gives size info, but requires a callback function for EnumDisplayMonitors  https://docs.microsoft.com/en-us/windows/desktop/api/Winuser/nf-winuser-getmonitorinfoa https://docs.microsoft.com/en-us/windows/desktop/api/Winuser/nf-winuser-enumdisplaymonitors

	scNms := make([]string, 0, 5)
	scno := 0
	mainsc := 0
	for {
		ok := _EnumDisplayDevices(0, uint32(scno), &ddinfo, 0)
		if !ok {
			break
		}
		if ddinfo.StateFlags&_DISPLAY_DEVICE_ACTIVE == 0 {
			scno++
			continue
		}
		nm := NullTermToString(ddinfo.DeviceName[:])
		nm = strings.TrimPrefix(nm, `\\.\`)
		scNms = append(scNms, nm)
		if ddinfo.StateFlags&_DISPLAY_DEVICE_PRIMARY_DEVICE != 0 {
			mainsc = len(scNms) - 1
		}
		scno++
	}

	app.screens = make([]*oswin.Screen, 1, len(scNms))
	sc := &oswin.Screen{Name: scNms[mainsc]}
	app.screens[0] = sc

	dc, _ := _GetDC(0) // 0 = for entire virtual screen
	defer _ReleaseDC(0, dc)

	widthPx := _GetDeviceCaps(dc, _HORZRES)
	heightPx := _GetDeviceCaps(dc, _VERTRES)
	widthMM := _GetDeviceCaps(dc, _HORZSIZE)
	heightMM := _GetDeviceCaps(dc, _VERTSIZE)
	dpi := float32(_GetDeviceCaps(dc, _LOGPIXELSX))

	// fmt.Printf("scrn dc logpixelsx: %v\n", dpi)
	depth := 32
	pixratio := float32(1.0)

	sc.ScreenNumber = 0
	sc.Geometry = image.Rectangle{Min: image.ZP, Max: image.Point{int(widthPx), int(heightPx)}}
	sc.Depth = depth
	sc.LogicalDPI = dpi // oswin.LogicalFmPhysicalDPI(dpi)
	sc.PhysicalDPI = dpi
	sc.DevicePixelRatio = pixratio
	sc.PhysicalSize = image.Point{int(widthMM), int(heightMM)}
}

func (app *appImpl) NScreens() int {
	return len(app.screens)
}

func (app *appImpl) Screen(scrN int) *oswin.Screen {
	sz := len(app.screens)
	if scrN < sz {
		return app.screens[scrN]
	}
	return nil
}

func (app *appImpl) ScreenByName(name string) *oswin.Screen {
	for _, sc := range app.screens {
		if sc.Name == name {
			return sc
		}
	}
	return nil
}

func (app *appImpl) NWindows() int {
	return len(app.winlist)
}

func (app *appImpl) Window(win int) oswin.Window {
	sz := len(app.winlist)
	if win < sz {
		return app.winlist[win]
	}
	return nil
}

func (app *appImpl) WindowByName(name string) oswin.Window {
	for _, win := range app.winlist {
		if win.Name() == name {
			return win
		}
	}
	return nil
}

func (app *appImpl) WindowInFocus() oswin.Window {
	for _, win := range app.winlist {
		if win.IsFocus() {
			return win
		}
	}
	return nil
}

func (app *appImpl) ContextWindow() oswin.Window {
	app.mu.Lock()
	cw := app.ctxtwin
	app.mu.Unlock()
	return cw
}

func (app *appImpl) Platform() oswin.Platforms {
	return oswin.Windows
}

func (app *appImpl) Name() string {
	return app.name
}

func (app *appImpl) SetName(name string) {
	app.name = name
}

func (app *appImpl) PrefsDir() string {
	// todo: could use a more official windows protocol to get this stuff..
	// https://msdn.microsoft.com/en-us/library/bb762188%28VS.85%29.aspx
	// with FOLDERID_RoamingAppData
	// https://stackoverflow.com/questions/6883779/what-are-the-best-practices-for-storing-user-preferences-and-settings-in-win32-d
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "AppData", "Roaming")
	// todo: convention is "manufacturer" before app -- not sure what that means in this context -- "Go"?
}

func (app *appImpl) GoGiPrefsDir() string {
	pdir := filepath.Join(app.PrefsDir(), "GoGi")
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (app *appImpl) AppPrefsDir() string {
	pdir := filepath.Join(app.PrefsDir(), app.Name())
	os.MkdirAll(pdir, 0755)
	return pdir
}

func (app *appImpl) FontPaths() []string {
	return []string{"C:\\Windows\\Fonts"}
}

func (app *appImpl) ClipBoard(win oswin.Window) clip.Board {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theClip
}

func (app *appImpl) Cursor(win oswin.Window) cursor.Cursor {
	app.mu.Lock()
	app.ctxtwin = win.(*windowImpl)
	app.mu.Unlock()
	return &theCursor
}

func (app *appImpl) About() string {
	return app.about
}

func (app *appImpl) SetAbout(about string) {
	app.about = about
}

func (app *appImpl) OpenURL(url string) {
	cmd := exec.Command("explorer", url)
	cmd.Run()
}

func (app *appImpl) SetQuitReqFunc(fun func()) {
	app.quitReqFunc = fun
}

func (app *appImpl) SetQuitCleanFunc(fun func()) {
	app.quitCleanFunc = fun
}

func (app *appImpl) QuitReq() {
	if app.quitting {
		return
	}
	if app.quitReqFunc != nil {
		app.quitReqFunc()
	} else {
		app.Quit()
	}
}

func (app *appImpl) IsQuitting() bool {
	return app.quitting
}

func (app *appImpl) QuitClean() {
	app.quitting = true
	if app.quitCleanFunc != nil {
		app.quitCleanFunc()
	}
	nwin := len(app.winlist)
	for i := nwin - 1; i >= 0; i-- {
		win := app.winlist[i]
		go win.Close()
	}
	for i := 0; i < nwin; i++ {
		<-app.quitCloseCnt
		// fmt.Printf("win closed: %v\n", i)
	}
}

func (app *appImpl) Quit() {
	app.QuitClean()
	SendAppMessage(msgQuit, 0, 0)
}

func sendQuit(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr) {
	theApp.QuitClean()
	return 0
}

//////////////////////////////////////////////////////////////////
//   Windows utilities

////////////////////////////////////////////////////////
// appWND is the handle to the "AppWindow".  The window encapsulates all
// oswin.Window operations in an actual Windows window so they all run on the
// main thread.  Since any messages sent to a window will be executed on the
// main thread, we can safely use the messages below.
var appHWND syscall.Handle

const (
	msgCreateWindow = _WM_USER + iota
	msgDeleteWindow
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

var appMsgs = map[uint32]func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) (lResult uintptr){}

func AddAppMsg(fn func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr)) uint32 {
	uMsg := currentUserWM.next()
	appMsgs[uMsg] = func(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
		fn(hwnd, uMsg, wParam, lParam)
		return 0
	}
	return uMsg
}

var mainCallback func(oswin.App)

func appWindowWndProc(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	switch uMsg {
	case msgCreateWindow:
		p := (*newWindowParams)(unsafe.Pointer(lParam))
		p.w, p.err = newWindow(p.opts)
	case msgDeleteWindow:
		hwnd := (syscall.Handle)(unsafe.Pointer(lParam))
		deleteWindow(hwnd)
	case msgMainCallback:
		go func() {
			mainCallback(theApp)
			SendAppMessage(msgQuit, 0, 0)
		}()
	case msgQuit:
		_PostQuitMessage(0)
	}
	fn := appMsgs[uMsg]
	if fn != nil {
		return fn(hwnd, uMsg, wParam, lParam)
	}
	return _DefWindowProc(hwnd, uMsg, wParam, lParam)
}

//go:uintptrescapes

func SendAppMessage(uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	return SendMessage(appHWND, uMsg, wParam, lParam)
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
	err := _GetWindowRect(appHWND, &wr)
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
	hDefaultCursor = _LoadCursor(0, _IDC_ARROW)
	// TODO(andlabs) hThisInstance
	return nil
}

//go:uintptrescapes

func SendMessage(hwnd syscall.Handle, uMsg uint32, wParam uintptr, lParam uintptr) (lResult uintptr) {
	return _sendMessage(hwnd, uMsg, wParam, lParam)
}
