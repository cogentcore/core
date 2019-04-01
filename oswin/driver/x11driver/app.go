// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build linux,!android dragonfly openbsd
// +build !3d

package x11driver

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/render"
	"github.com/BurntSushi/xgb/shm"
	"github.com/BurntSushi/xgb/xproto"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/clip"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki/bitflag"
	"golang.org/x/image/math/f64"
)

// TODO: check that xgb is safe to use concurrently from multiple goroutines.
// For example, its Conn.WaitForEvent concept is a method, not a channel, so
// it's not obvious how to interrupt it to service a NewWindow request.

type appImpl struct {
	xc      *xgb.Conn
	xsi     *xproto.SetupInfo
	xsci    *xproto.ScreenInfo
	keysyms KeysymTable

	atomNETWMName       xproto.Atom
	atomUTF8String      xproto.Atom
	atomWMDeleteWindow  xproto.Atom
	atomWMProtocols     xproto.Atom
	atomWMTakeFocus     xproto.Atom
	atomWMChangeState   xproto.Atom
	atomNetFrameExtents xproto.Atom
	atomNetActiveWindow xproto.Atom
	atomClipboardSel    xproto.Atom
	atomPrimarySel      xproto.Atom
	atomTargets         xproto.Atom
	atomMultiple        xproto.Atom
	atomTimestamp       xproto.Atom
	atomIncr            xproto.Atom
	atomText            xproto.Atom

	pixelsPerPt  float32
	pictformat24 render.Pictformat
	pictformat32 render.Pictformat

	// window32 and its related X11 resources is an unmapped window so that we
	// have a depth-32 window to create depth-32 pixmaps from, i.e. pixmaps
	// with an alpha channel. The root window isn't guaranteed to be depth-32.
	gcontext32 xproto.Gcontext
	window32   xproto.Window

	// opaqueP is a fully opaque, solid fill picture.
	opaqueP render.Picture

	uniformMu sync.Mutex
	uniformC  render.Color
	uniformP  render.Picture

	mu              sync.Mutex
	images          map[shm.Seg]*imageImpl
	uploads         map[uint16]chan struct{}
	windows         map[xproto.Window]*windowImpl
	winlist         []*windowImpl
	screens         []*oswin.Screen
	ctxtwin         *windowImpl
	nPendingUploads int
	completionKeys  []uint16
	selNotifyChan   chan xproto.SelectionNotifyEvent
	name            string
	about           string
	quitting        bool // set to true when quitting and closing windows
	quitEndRun      bool
	quitCloseCnt    chan struct{} // counts windows to make sure all are closed before done
	quitReqFunc     func()
	quitCleanFunc   func()
}

var theApp *appImpl

func newAppImpl(xc *xgb.Conn) (*appImpl, error) {
	app := &appImpl{
		xc:            xc,
		xsi:           xproto.Setup(xc),
		images:        map[shm.Seg]*imageImpl{},
		uploads:       map[uint16]chan struct{}{},
		windows:       map[xproto.Window]*windowImpl{},
		winlist:       make([]*windowImpl, 0),
		selNotifyChan: make(chan xproto.SelectionNotifyEvent, 100),
		quitCloseCnt:  make(chan struct{}),
		name:          "GoGi",
	}
	app.xsci = app.xsi.DefaultScreen(xc)
	if err := app.initAtoms(); err != nil {
		return nil, err
	}
	if err := app.initKeyboardMapping(); err != nil {
		return nil, err
	}
	const (
		mmPerInch = 25.4
		ptPerInch = 72
	)
	pixelsPerMM := float32(app.xsci.WidthInPixels) / float32(app.xsci.WidthInMillimeters)
	app.pixelsPerPt = pixelsPerMM * mmPerInch / ptPerInch
	if err := app.initPictformats(); err != nil {
		return nil, err
	}
	if err := app.initWindow32(); err != nil {
		return nil, err
	}

	var err error
	app.opaqueP, err = render.NewPictureId(xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewPictureId failed: %v", err)
	}
	app.uniformP, err = render.NewPictureId(xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewPictureId failed: %v", err)
	}
	render.CreateSolidFill(app.xc, app.opaqueP, render.Color{
		Red:   0xffff,
		Green: 0xffff,
		Blue:  0xffff,
		Alpha: 0xffff,
	})
	render.CreateSolidFill(app.xc, app.uniformP, render.Color{})

	nsc := len(app.xsi.Roots)
	if nsc == 0 {
		log.Printf("X11 startup: No Root screens found\n")
		nsc = 1
	}

	// note: putting default screen first, but this then makes rest of screens out of order
	// relative to xwindows's list.
	app.screens = make([]*oswin.Screen, nsc)
	sc := &oswin.Screen{}
	app.screens[0] = sc

	widthPx := app.xsci.WidthInPixels
	heightPx := app.xsci.HeightInPixels
	widthMM := app.xsci.WidthInMillimeters
	heightMM := app.xsci.WidthInMillimeters

	dpi := 25.4 * (float32(widthPx) / float32(widthMM))
	pixratio := float32(1.0)

	sc.ScreenNumber = 0
	sc.Geometry = image.Rectangle{Min: image.ZP, Max: image.Point{int(widthPx), int(heightPx)}}
	sc.Depth = int(app.xsci.RootDepth)
	sc.LogicalDPI = dpi
	sc.PhysicalDPI = dpi
	sc.DevicePixelRatio = pixratio
	sc.PhysicalSize = image.Point{int(widthMM), int(heightMM)}
	sc.Name = app.xsi.Vendor + ":0"

	sidx := 1
	for si := 0; si < nsc; si++ {
		if si == app.xc.DefaultScreen {
			continue
		}
		sci := &app.xsi.Roots[si]

		sc := &oswin.Screen{}
		app.screens[sidx] = sc

		widthPx := sci.WidthInPixels
		heightPx := sci.HeightInPixels
		widthMM := sci.WidthInMillimeters
		heightMM := sci.WidthInMillimeters

		dpi := 25.4 * (float32(widthPx) / float32(widthMM))
		pixratio := float32(1.0)

		sc.ScreenNumber = 0
		sc.Geometry = image.Rectangle{Min: image.ZP, Max: image.Point{int(widthPx), int(heightPx)}}
		sc.Depth = int(sci.RootDepth)
		sc.LogicalDPI = dpi
		sc.PhysicalDPI = dpi
		sc.DevicePixelRatio = pixratio
		sc.PhysicalSize = image.Point{int(widthMM), int(heightMM)}
		sc.Name = fmt.Sprintf("%v:%v", app.xsi.Vendor, sidx)

		sidx++
	}

	oswin.TheApp = app
	theApp = app

	go app.run()
	return app, nil
}

func (app *appImpl) run() {
mainloop:
	for {
		// fmt.Printf("wait..\n")
		ev, err := app.xc.WaitForEvent()
		// fmt.Printf("got..\n")
		if app.quitEndRun {
			break mainloop
		}
		if err != nil {
			log.Printf("x11driver: xproto.WaitForEvent: %v", err)
			continue
		}
		noWindowFound := false
		switch ev := ev.(type) {
		case xproto.DestroyNotifyEvent:
			if w := app.findWindow(ev.Window); w != nil {
				w.closed()
			}
		case shm.CompletionEvent:
			app.mu.Lock()
			app.completionKeys = append(app.completionKeys, ev.Sequence)
			app.handleCompletions()
			app.mu.Unlock()

		case xproto.ClientMessageEvent:
			if ev.Type != app.atomWMProtocols || ev.Format != 32 {
				break
			}
			switch xproto.Atom(ev.Data.Data32[0]) {
			case app.atomWMDeleteWindow:
				if w := app.findWindow(ev.Window); w != nil {
					w.CloseReq()
				} else {
					noWindowFound = true
				}
			case app.atomWMTakeFocus:
				xproto.SetInputFocus(app.xc, xproto.InputFocusParent, ev.Window, xproto.Timestamp(ev.Data.Data32[1]))
			}

		case xproto.ConfigureNotifyEvent:
			if w := app.findWindow(ev.Window); w != nil {
				w.handleConfigureNotify(ev)
			} else {
				noWindowFound = true
			}

		case xproto.ExposeEvent:
			if w := app.findWindow(ev.Window); w != nil {
				// A non-zero Count means that there are more expose events
				// coming. For example, a non-rectangular exposure (e.g. from a
				// partially overlapped window) will result in multiple expose
				// events whose dirty rectangles combine to define the dirty
				// region. Go's paint events do not provide dirty regions, so
				// we only pass on the final X11 expose event.
				if ev.Count == 0 {
					w.handleExpose()
				}
			} else {
				noWindowFound = true
			}

		case xproto.FocusInEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.mu.Lock()
				bitflag.Clear(&w.Flag, int(oswin.Minimized))
				bitflag.Set(&w.Flag, int(oswin.Focus))
				w.mu.Unlock()
				// fmt.Printf("focused %v\n", w.Name())
				sendWindowEvent(w, window.Focus)
			} else {
				noWindowFound = true
			}

		case xproto.FocusOutEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.mu.Lock()
				bitflag.Clear(&w.Flag, int(oswin.Focus))
				w.mu.Unlock()
				// fmt.Printf("defocused %v\n", w.Name())
				sendWindowEvent(w, window.DeFocus)
			} else {
				noWindowFound = true
			}

		case xproto.KeyPressEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.handleKey(ev.Detail, ev.State, key.Press)
			} else {
				noWindowFound = true
			}

		case xproto.KeyReleaseEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.handleKey(ev.Detail, ev.State, key.Release)
			} else {
				noWindowFound = true
			}

		case xproto.ButtonPressEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.handleMouse(ev.EventX, ev.EventY, ev.Detail, ev.State, mouse.Press)
			} else {
				noWindowFound = true
			}

		case xproto.ButtonReleaseEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.handleMouse(ev.EventX, ev.EventY, ev.Detail, ev.State, mouse.Release)
			} else {
				noWindowFound = true
			}

		case xproto.MotionNotifyEvent:
			if w := app.findWindow(ev.Event); w != nil {
				w.handleMouse(ev.EventX, ev.EventY, 0, ev.State, mouse.NoAction)
			} else {
				noWindowFound = true
			}

		case xproto.SelectionNotifyEvent:
			app.selNotifyChan <- ev

		case xproto.SelectionRequestEvent:
			theClip.SendLastWrite(ev)
		}

		if noWindowFound { // we expect this actually
			// log.Printf("x11driver: no window found for event %T", ev)
		}
	}
	fmt.Printf("out of event loop\n")
}

// TODO: is findImage and the app.images field unused? Delete?

func (app *appImpl) findImage(key shm.Seg) *imageImpl {
	app.mu.Lock()
	b := app.images[key]
	app.mu.Unlock()
	return b
}

func (app *appImpl) findWindow(key xproto.Window) *windowImpl {
	app.mu.Lock()
	w := app.windows[key]
	app.mu.Unlock()
	return w
}

// handleCompletions must only be called while holding app.mu.
func (app *appImpl) handleCompletions() {
	if app.nPendingUploads != 0 {
		return
	}
	for _, ck := range app.completionKeys {
		completion, ok := app.uploads[ck]
		if !ok {
			log.Printf("x11driver: no matching upload for a SHM completion event")
			continue
		}
		delete(app.uploads, ck)
		close(completion)
	}
	app.completionKeys = app.completionKeys[:0]
}

const (
	maxShmSide = 0x00007fff // 32,767 pixels.
	maxShmSize = 0x10000000 // 268,435,456 bytes.
)

func (app *appImpl) NewImage(size image.Point) (retBuf oswin.Image, retErr error) {
	// TODO: detect if the X11 server or connection cannot support SHM pixmaps,
	// and fall back to regular pixmaps.

	w, h := int64(size.X), int64(size.Y)
	if w < 0 || maxShmSide < w || h < 0 || maxShmSide < h || maxShmSize < 4*w*h {
		return nil, fmt.Errorf("x11driver: invalid image size %v", size)
	}

	b := &imageImpl{
		app: app,
		rgba: image.RGBA{
			Stride: 4 * size.X,
			Rect:   image.Rectangle{Max: size},
		},
		size: size,
	}

	if size.X == 0 || size.Y == 0 {
		// No-op, but we can't take the else path because the minimum shmget
		// size is 1.
	} else {
		xs, err := shm.NewSegId(app.xc)
		if err != nil {
			return nil, fmt.Errorf("x11driver: shm.NewSegId: %v", err)
		}

		bufLen := 4 * size.X * size.Y
		shmid, addr, err := shmOpen(bufLen)
		if err != nil {
			return nil, fmt.Errorf("x11driver: shmOpen: %v", err)
		}
		defer func() {
			if retErr != nil {
				shmClose(addr)
			}
		}()
		a := (*[maxShmSize]byte)(addr)
		b.buf = (*a)[:bufLen:bufLen]
		b.rgba.Pix = b.buf
		b.addr = addr

		// readOnly is whether the shared memory is read-only from the X11 server's
		// point of view. We need false to use SHM pixmaps.
		const readOnly = false
		shm.Attach(app.xc, xs, uint32(shmid), readOnly)
		b.xs = xs
	}

	app.mu.Lock()
	app.images[b.xs] = b
	app.mu.Unlock()

	return b, nil
}

func (app *appImpl) NewTexture(win oswin.Window, size image.Point) (oswin.Texture, error) {
	w, h := int64(size.X), int64(size.Y)
	if w < 0 || maxShmSide < w || h < 0 || maxShmSide < h || maxShmSize < 4*w*h {
		return nil, fmt.Errorf("x11driver: invalid texture size %v", size)
	}
	if w == 0 || h == 0 {
		return &textureImpl{
			app:  app,
			size: size,
		}, nil
	}

	xm, err := xproto.NewPixmapId(app.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewPixmapId failed: %v", err)
	}
	xp, err := render.NewPictureId(app.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewPictureId failed: %v", err)
	}
	xproto.CreatePixmap(app.xc, textureDepth, xm, xproto.Drawable(app.window32), uint16(w), uint16(h))
	render.CreatePicture(app.xc, xp, xproto.Drawable(xm), app.pictformat32, render.CpRepeat, []uint32{render.RepeatPad})
	render.SetPictureFilter(app.xc, xp, uint16(len("bilinear")), "bilinear", nil)
	// The X11 server doesn't zero-initialize the pixmap. We do it ourselves.
	render.FillRectangles(app.xc, render.PictOpSrc, xp, render.Color{}, []xproto.Rectangle{{
		Width:  uint16(w),
		Height: uint16(h),
	}})

	ww := win.(*windowImpl)

	nt := &textureImpl{
		w:    ww,
		app:  app,
		size: size,
		xm:   xm,
		xp:   xp,
	}
	ww.AddTexture(nt)
	return nt, nil
}

// borderwidth doesn't seem to actually do anything in ubuntu or xfce
var WindowBorderWidth = 0

func (app *appImpl) NewWindow(opts *oswin.NewWindowOptions) (oswin.Window, error) {
	if opts == nil {
		opts = &oswin.NewWindowOptions{}
	}
	opts.Fixup()
	// can also apply further tuning here..

	xw, err := xproto.NewWindowId(app.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewWindowId failed: %v", err)
	}
	xg, err := xproto.NewGcontextId(app.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: xproto.NewGcontextId failed: %v", err)
	}
	xp, err := render.NewPictureId(app.xc)
	if err != nil {
		return nil, fmt.Errorf("x11driver: render.NewPictureId failed: %v", err)
	}
	pictformat := render.Pictformat(0)
	switch app.xsci.RootDepth {
	default:
		return nil, fmt.Errorf("x11driver: unsupported root depth %d", app.xsci.RootDepth)
	case 24:
		pictformat = app.pictformat24
	case 32:
		pictformat = app.pictformat32
	}

	// todo: multiple screens..
	sc := app.Screen(0)
	dpi := sc.PhysicalDPI
	ldpi := sc.LogicalDPI

	w := &windowImpl{
		app:     app,
		xw:      xw,
		xg:      xg,
		xp:      xp,
		xevents: make(chan xgb.Event),
		WindowBase: oswin.WindowBase{
			Pos:     opts.Pos,
			PhysDPI: dpi,
			LogDPI:  ldpi,
		},
	}

	app.mu.Lock()
	app.windows[xw] = w
	app.winlist = append(app.winlist, w)
	app.mu.Unlock()

	if opts.Pos.X < 40 {
		opts.Pos.X = 40
	}
	if opts.Pos.Y < 40 {
		opts.Pos.Y = 40
	}

	xproto.CreateWindow(app.xc, app.xsci.RootDepth, xw, app.xsci.Root,
		int16(opts.Pos.X), int16(opts.Pos.Y), uint16(opts.Size.X), uint16(opts.Size.Y), uint16(WindowBorderWidth),
		xproto.WindowClassInputOutput, app.xsci.RootVisual,
		xproto.CwEventMask,
		[]uint32{0 |
			xproto.EventMaskKeyPress |
			xproto.EventMaskKeyRelease |
			xproto.EventMaskButtonPress |
			xproto.EventMaskButtonRelease |
			xproto.EventMaskPointerMotion |
			xproto.EventMaskExposure |
			xproto.EventMaskStructureNotify |
			xproto.EventMaskFocusChange,
		},
	)
	app.setProperty(xw, app.atomWMProtocols, app.atomWMDeleteWindow, app.atomWMTakeFocus)

	// fmt.Printf("create pos: %v\n", opts.Pos)
	// todo: opts
	// dialog, modal, tool, fullscreen := oswin.WindowFlagsToBool(opts.Flags)

	title := []byte(opts.GetTitle())
	xproto.ChangeProperty(app.xc, xproto.PropModeReplace, xw, app.atomNETWMName, app.atomUTF8String, 8, uint32(len(title)), title)

	xproto.CreateGC(app.xc, xg, xproto.Drawable(xw), 0, nil)
	render.CreatePicture(app.xc, xp, xproto.Drawable(xw), pictformat, 0, nil)

	xproto.MapWindow(app.xc, xw)

	if opts.Pos != image.ZP {
		w.SetGeom(opts.Pos, opts.Size)
	}

	return w, nil
}

func (app *appImpl) initAtoms() (err error) {
	app.atomNETWMName, err = app.internAtom("_NET_WM_NAME")
	if err != nil {
		return err
	}
	app.atomUTF8String, err = app.internAtom("UTF8_STRING")
	if err != nil {
		return err
	}
	app.atomWMDeleteWindow, err = app.internAtom("WM_DELETE_WINDOW")
	if err != nil {
		return err
	}
	app.atomWMProtocols, err = app.internAtom("WM_PROTOCOLS")
	if err != nil {
		return err
	}
	app.atomWMTakeFocus, err = app.internAtom("WM_TAKE_FOCUS")
	if err != nil {
		return err
	}
	app.atomWMChangeState, err = app.internAtom("WM_CHANGE_STATE")
	if err != nil {
		return err
	}
	app.atomNetFrameExtents, err = app.internAtom("_NET_FRAME_EXTENTS")
	if err != nil {
		return err
	}
	app.atomNetActiveWindow, err = app.internAtom("_NET_ACTIVE_WINDOW")
	if err != nil {
		return err
	}
	app.atomClipboardSel, err = app.internAtom("CLIPBOARD")
	if err != nil {
		return err
	}
	app.atomPrimarySel, err = app.internAtom("PRIMARY")
	if err != nil {
		return err
	}
	app.atomTargets, err = app.internAtom("TARGETS")
	if err != nil {
		return err
	}
	app.atomMultiple, err = app.internAtom("MULTIPLE")
	if err != nil {
		return err
	}
	app.atomTimestamp, err = app.internAtom("TIMESTAMP")
	if err != nil {
		return err
	}
	app.atomIncr, err = app.internAtom("INCR")
	if err != nil {
		return err
	}
	app.atomText, err = app.internAtom("TEXT")
	if err != nil {
		return err
	}
	return nil
}

func (app *appImpl) internAtom(name string) (xproto.Atom, error) {
	r, err := xproto.InternAtom(app.xc, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("x11driver: xproto.InternAtom failed: %v", err)
	}
	if r == nil {
		return 0, fmt.Errorf("x11driver: xproto.InternAtom failed")
	}
	return r.Atom, nil
}

func (app *appImpl) initKeyboardMapping() error {
	const keyLo, keyHi = 8, 255
	km, err := xproto.GetKeyboardMapping(app.xc, keyLo, keyHi-keyLo+1).Reply()
	if err != nil {
		return err
	}
	n := int(km.KeysymsPerKeycode)
	if n < 2 {
		return fmt.Errorf("x11driver: too few keysyms per keycode: %d", n)
	}
	for i := keyLo; i <= keyHi; i++ {
		app.keysyms[i][0] = uint32(km.Keysyms[(i-keyLo)*n+0])
		app.keysyms[i][1] = uint32(km.Keysyms[(i-keyLo)*n+1])
	}
	return nil
}

func (app *appImpl) initPictformats() error {
	pformats, err := render.QueryPictFormats(app.xc).Reply()
	if err != nil {
		return fmt.Errorf("x11driver: render.QueryPictFormats failed: %v", err)
	}
	app.pictformat24, err = findPictformat(pformats.Formats, 24)
	if err != nil {
		return err
	}
	app.pictformat32, err = findPictformat(pformats.Formats, 32)
	if err != nil {
		return err
	}
	return nil
}

func findPictformat(fs []render.Pictforminfo, depth byte) (render.Pictformat, error) {
	// This presumes little-endian BGRA.
	want := render.Directformat{
		RedShift:   16,
		RedMask:    0xff,
		GreenShift: 8,
		GreenMask:  0xff,
		BlueShift:  0,
		BlueMask:   0xff,
		AlphaShift: 24,
		AlphaMask:  0xff,
	}
	if depth == 24 {
		want.AlphaShift = 0
		want.AlphaMask = 0x00
	}
	for _, f := range fs {
		if f.Type == render.PictTypeDirect && f.Depth == depth && f.Direct == want {
			return f.Id, nil
		}
	}
	return 0, fmt.Errorf("x11driver: no matching Pictformat for depth %d", depth)
}

func (app *appImpl) initWindow32() error {
	visualid, err := findVisual(app.xsci, 32)
	if err != nil {
		return err
	}
	colormap, err := xproto.NewColormapId(app.xc)
	if err != nil {
		return fmt.Errorf("x11driver: xproto.NewColormapId failed: %v", err)
	}
	if err := xproto.CreateColormapChecked(
		app.xc, xproto.ColormapAllocNone, colormap, app.xsci.Root, visualid).Check(); err != nil {
		return fmt.Errorf("x11driver: xproto.CreateColormap failed: %v", err)
	}
	app.window32, err = xproto.NewWindowId(app.xc)
	if err != nil {
		return fmt.Errorf("x11driver: xproto.NewWindowId failed: %v", err)
	}
	app.gcontext32, err = xproto.NewGcontextId(app.xc)
	if err != nil {
		return fmt.Errorf("x11driver: xproto.NewGcontextId failed: %v", err)
	}
	const depth = 32
	xproto.CreateWindow(app.xc, depth, app.window32, app.xsci.Root,
		0, 0, 1, 1, 0,
		xproto.WindowClassInputOutput, visualid,
		// The CwBorderPixel attribute seems necessary for depth == 32. See
		// http://stackoverflow.com/questions/3645632/how-to-create-a-window-with-a-bit-depth-of-32
		xproto.CwBorderPixel|xproto.CwColormap,
		[]uint32{0, uint32(colormap)},
	)
	xproto.CreateGC(app.xc, app.gcontext32, xproto.Drawable(app.window32), 0, nil)
	return nil
}

func findVisual(xsci *xproto.ScreenInfo, depth byte) (xproto.Visualid, error) {
	for _, d := range xsci.AllowedDepths {
		if d.Depth != depth {
			continue
		}
		for _, v := range d.Visuals {
			if v.RedMask == 0xff0000 && v.GreenMask == 0xff00 && v.BlueMask == 0xff {
				return v.VisualId, nil
			}
		}
	}
	return 0, fmt.Errorf("x11driver: no matching Visualid")
}

func (app *appImpl) setProperty(xw xproto.Window, prop xproto.Atom, values ...xproto.Atom) {
	b := make([]byte, len(values)*4)
	for i, v := range values {
		b[4*i+0] = uint8(v >> 0)
		b[4*i+1] = uint8(v >> 8)
		b[4*i+2] = uint8(v >> 16)
		b[4*i+3] = uint8(v >> 24)
	}
	xproto.ChangeProperty(app.xc, xproto.PropModeReplace, xw, prop, xproto.AtomAtom, 32, uint32(len(values)), b)
}

func (app *appImpl) drawUniform(xp render.Picture, src2dst *f64.Aff3, src color.Color, sr image.Rectangle, op draw.Op, opts *oswin.DrawOptions) {
	if sr.Empty() {
		return
	}

	if opts == nil && *src2dst == (f64.Aff3{1, 0, 0, 0, 1, 0}) {
		fill(app.xc, xp, sr, src, op)
		return
	}

	r, g, b, a := src.RGBA()
	c := render.Color{
		Red:   uint16(r),
		Green: uint16(g),
		Blue:  uint16(b),
		Alpha: uint16(a),
	}
	points := trifanPoints(src2dst, sr)

	app.uniformMu.Lock()
	defer app.uniformMu.Unlock()

	if app.uniformC != c {
		app.uniformC = c
		render.FreePicture(app.xc, app.uniformP)
		render.CreateSolidFill(app.xc, app.uniformP, c)
	}

	if op == draw.Src {
		// We implement draw.Src as render.PictOpOutReverse followed by
		// render.PictOpOver, for the same reason as in textureImpl.draw.
		render.TriFan(app.xc, render.PictOpOutReverse, app.opaqueP, xp, 0, 0, 0, points[:])
	}
	render.TriFan(app.xc, render.PictOpOver, app.uniformP, xp, 0, 0, 0, points[:])
}

func (app *appImpl) DeleteWin(id xproto.Window) {
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
	return oswin.LinuxX11
}

func (app *appImpl) Name() string {
	return app.name
}

func (app *appImpl) SetName(name string) {
	app.name = name
}

func (app *appImpl) PrefsDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, ".config")
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
	return []string{"/usr/share/fonts/truetype"}
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
	cmd := exec.Command("xdg-open", url)
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
	app.quitEndRun = true

	vdat := []uint32{1, xproto.TimeCurrentTime, 0, 0, 0} // 1 = make it active somehow
	dat := xproto.ClientMessageDataUnionData32New(vdat)

	// we just send ourselves a dummy message so the event loop gets something
	minmsg := xproto.ClientMessageEvent{
		Sequence: 0, // no idea what this is..
		Format:   32,
		Window:   app.window32,
		Type:     app.atomUTF8String, // whatever
		Data:     dat,
	}
	mask := xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify
	xproto.SendEvent(app.xc, true, app.window32, uint32(mask), string(minmsg.Bytes()))

}
