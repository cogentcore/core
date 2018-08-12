// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package windriver

import (
	"fmt"
	"image"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/driver/internal/win32"
)

var theApp = &appImpl{
	windows: make(map[syscall.Handle]*windowImpl),
	winlist: make([]*windowImpl, 0),
	screens: make([]*oswin.Screen, 0),
	name:    "GoGi",
}

type appImpl struct {
	mu      sync.Mutex
	windows map[syscall.Handle]*windowImpl
	winlist []*windowImpl
	screens []*oswin.Screen
	name    string
	about   string
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
	return newTexture(size)
}

func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	if opts == nil {
		opts = &oswin.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	// todo: multiple screens..
	sc := app.Screen(0)
	dpi := sc.PhysicalDPI
	ldpi := oswin.LogicalFmPhysicalDPI(dpi)

	w := &windowImpl{
		WindowBase: oswin.WindowBase{
			Pos:     opts.Pos,
			PhysDPI: dpi,
			LogDPI:  ldpi,
		},
	}

	var err error
	w.hwnd, err = win32.NewWindow(opts)
	if err != nil {
		return nil, err
	}

	app.mu.Lock()
	app.windows[w.hwnd] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	err = win32.ResizeClientRect(w.hwnd, opts.Size)
	if err != nil {
		return nil, err
	}

	win32.Show(w.hwnd)

	if procGetDpiForWindow.Find() == nil { // has it
		dpi = float32(_GetDpiForWindow(w.hwnd))
		ldpi = oswin.LogicalFmPhysicalDPI(dpi)
		w.PhysDPI = dpi
		w.LogDPI = ldpi
		if sc.PhysicalDPI == 96 {
			sc.PhysicalDPI = dpi
			sc.LogicalDPI = ldpi
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

func (app *appImpl) initScreens() {
	// https://blogs.windows.com/buildingapps/2016/10/24/high-dpi-scaling-improvements-for-desktop-applications-and-mixed-mode-dpi-scaling-in-the-windows-10-anniversary-update/

	// https://github.com/Microsoft/Windows-classic-samples/tree/master/Samples/DPIAwarenessPerWindow/cpp
	// https://msdn.microsoft.com/en-us/library/windows/desktop/hh447398(v=vs.85).aspx

	// for now, just gonna fake it..
	app.screens = make([]*oswin.Screen, 1)
	sc := &oswin.Screen{}
	app.screens[0] = sc

	_SetProcessDpiAwareness(_PROCESS_PER_MONITOR_DPI_AWARE)

	// todo: this is not working at all!
	widthPx, heightPx := win32.ScreenSize()

	if widthPx == 0 {
		widthPx = 1200
	}
	if heightPx == 0 {
		heightPx = 800
	}

	// widthMM := app.xsi.WidthInMillimeters
	// heightMM := app.xsi.WidthInMillimeters
	// dpi := 25.4 * (float32(widthPx) / float32(widthMM))

	dpi := float32(96)

	depth := 32
	pixratio := float32(1.0)

	sc.ScreenNumber = 0
	sc.Geometry = image.Rectangle{Min: image.ZP, Max: image.Point{int(widthPx), int(heightPx)}}
	sc.Depth = depth
	sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(dpi)
	sc.PhysicalDPI = dpi
	sc.DevicePixelRatio = pixratio
	sc.PhysicalSize = image.Point{int(widthPx), int(heightPx)} // don't know yet..
	// todo: rest of the fields
	//	fmt.Printf("screen: %+v\n", sc)
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

func (app *appImpl) ClipBoard() clip.Board {
	return &theClip
}

func (app *appImpl) Cursor() cursor.Cursor {
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
