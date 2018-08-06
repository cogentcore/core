// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/pprof"

	"github.com/chewxy/math32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/dnd"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/lifecycle"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/oswin/paint"
	"github.com/goki/gi/oswin/window"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
	"github.com/goki/prof"

	"time"
)

// EventSkipLagMSec is the number of milliseconds of lag between the time the
// event was sent to the time it is being processed, above which a repeated
// event type (scroll, drag, resize) is skipped
var EventSkipLagMSec = 50

// DragStartMSec is the number of milliseconds to wait before initiating a
// regular mouse drag event (as opposed to a basic mouse.Press)
var DragStartMSec = 50

// DragStartPix is the number of pixels that must be moved before
// initiating a regular mouse drag event (as opposed to a basic mouse.Press)
var DragStartPix = 4

// DNDStartMSec is the number of milliseconds to wait before initiating a
// drag-n-drop event -- gotta drag it like you mean it
var DNDStartMSec = 200

// DNDStartPix is the number of pixels that must be moved before
// initiating a drag-n-drop event -- gotta drag it like you mean it
var DNDStartPix = 20

// HoverStartMSec is the number of milliseconds to wait before initiating a
// hover event
var HoverStartMSec = 2000

// HoverMaxPix is the maximum number of pixels that mouse can move and still
// register a Hover event
var HoverMaxPix = 5

// notes: oswin/Image is the thing that a Vp should have uploader uploads the
// buffer/image to the window -- can also render directly onto window using
// textures using the drawer interface, but..

// Window provides an OS-specific window and all the associated event
// handling.  Widgets connect to event signals to receive relevant GUI events.
// There is a master Viewport that contains the full bitmap image of the
// window, onto which most widgets render.  For main windows (not dialogs or
// other popups), there is a master vertical layout under the Viewport
// (MasterVLay), whose first element is the MainMenu for the window (which can
// be empty, in which case it is not displayed).  On MacOS, this main menu is
// typically not directly visible, and instead updates the overall menubar.
type Window struct {
	NodeBase
	Title         string            `desc:"displayed name of window, for window manager etc -- window object name is the internal handle and is used for tracking property info etc"`
	OSWin         oswin.Window      `json:"-" xml:"-" view:"-" desc:"OS-specific window interface -- handles all the os-specific functions, including delivering events etc"`
	HasGeomPrefs  bool              `desc:"did this window have WinGeomPrefs setting that sized it -- affects whether other defauld geom should be applied"`
	Viewport      *Viewport2D       `json:"-" xml:"-" desc:"convenience pointer to window's master viewport child that handles the rendering"`
	MasterVLay    *Layout           `json:"-" xml:"-" desc:"main vertical layout under Viewport -- first element is MainMenu (always -- leave empty to not render)"`
	MainMenu      *MenuBar          `json:"-" xml:"-" desc:"main menu -- is first element of MasterVLay always -- leave empty to not render.  On MacOS, this drives screen main menu"`
	OverlayVp     Viewport2D        `json:"-" xml:"-" desc:"a separate collection of items to be rendered as overlays -- this viewport is cleared to transparent and all the elements in it are re-rendered if any of them needs to be updated -- generally each item should be manually positioned"`
	WinTex        oswin.Texture     `json:"-" xml:"-" view:"-" desc:"texture for the entire window -- all rendering is done onto this texture, which is then published into the window"`
	OverTexActive bool              `json:"-" xml:"-" desc:"is the overlay texture active and should be uploaded to window?"`
	OverTex       oswin.Texture     `json:"-" xml:"-" view:"-" desc:"overlay texture that is updated by OverlayVp viewport"`
	LastSelMode   mouse.SelectModes `json:"-" xml:"-" desc:"Last Select Mode from Mouse, Keyboard events"`
	Focus         ki.Ki             `json:"-" xml:"-" desc:"node receiving keyboard events"`
	DNDData       mimedata.Mimes    `json:"-" xml:"-" desc:"drag-n-drop data -- if non-nil, then DND is taking place"`
	DNDSource     ki.Ki             `json:"-" xml:"-" desc:"drag-n-drop source node"`
	DNDImage      ki.Ki             `json:"-" xml:"-" desc:"drag-n-drop node with image of source, that is actually dragged -- typically a Bitmap but can be anything (that renders in Overlay for 2D)"`
	DNDFinalEvent *dnd.Event        `json:"-" xml:"-" view:"-" desc:"final event for DND which is sent if a finalize is received"`
	DNDMod        dnd.DropMods      `json:"-" xml:"-" desc:"current DND modifier (Copy, Move, Link) -- managed by DNDSetCursor for updating cursor"`
	Dragging      ki.Ki             `json:"-" xml:"-" desc:"node receiving mouse dragging events -- not for DND but things like sliders"`
	Popup         ki.Ki             `jsom:"-" xml:"-" desc:"Current popup viewport that gets all events"`
	PopupStack    []ki.Ki           `jsom:"-" xml:"-" desc:"stack of popups"`
	FocusStack    []ki.Ki           `jsom:"-" xml:"-" desc:"stack of focus"`
	NextPopup     ki.Ki             `json:"-" xml:"-" desc:"this popup will be pushed at the end of the current event cycle"`
	DoFullRender  bool              `json:"-" xml:"-" desc:"triggers a full re-render of the window within the event loop -- cleared once done"`

	EventSigs     [oswin.EventTypeN][EventPrisN]ki.Signal `json:"-" xml:"-" view:"-" desc:"signals for communicating each type of event, organized by priority"`
	stopEventLoop bool
}

var KiT_Window = kit.Types.AddType(&Window{}, nil)

// EventPris for different queues of event signals, processed in priority order
type EventPris int32

const (
	// HiPri = high priority -- event receivers processed first -- can be used
	// to override default behavior
	HiPri EventPris = iota

	// RegPri = default regular priority -- most should be here
	RegPri

	// LowPri = low priority -- processed last -- typically for containers /
	// dialogs etc
	LowPri

	// LowRawPri = unfiltered (raw) low priority -- ignores whether the event
	// was already processed -- e.g., DoubleClick, Accept in dialog
	LowRawPri

	EventPrisN

	// AllPris = -1 = all priorities (for delete cases only)
	AllPris EventPris = -1
)

//go:generate stringer -type=EventPris

// NewWindow creates a new window with given internal name handle, display
// name, and options.
func NewWindow(name, title string, opts *oswin.NewWindowOptions) *Window {
	Init() // overall gogi system initialization
	win := &Window{}
	win.InitName(win, name)
	win.Title = title
	win.SetOnlySelfUpdate() // has its own PublishImage update logic
	var err error
	win.OSWin, err = oswin.TheApp.NewWindow(opts)
	if err != nil {
		fmt.Printf("GoGi NewWindow error: %v \n", err)
		return nil
	}
	win.WinTex, err = oswin.TheApp.NewTexture(win.OSWin, opts.Size) // note size will be in dots
	if err != nil {
		fmt.Printf("GoGi NewTexture error: %v \n", err)
		return nil
	}
	win.OSWin.SetName(title)
	win.OSWin.SetParent(win.This)
	win.NodeSig.Connect(win.This, SignalWindowPublish)
	return win
}

// NewWindow2D creates a new standard 2D window with given internal handle
// name, display name, and sizing, with default positioning, and initializes a
// 2D viewport within it -- stdPixels means use standardized "pixel" units for
// the display size (96 per inch), not the actual underlying raw display dot
// pixels.
func NewWindow2D(name, title string, width, height int, stdPixels bool) *Window {
	Init() // overall gogi system initialization
	opts := &oswin.NewWindowOptions{
		Title: title, Size: image.Point{width, height}, StdPixels: stdPixels,
	}
	wgp := WinGeomPrefs.Pref(name, nil)
	if wgp != nil {
		opts.Size = wgp.Size
		opts.Pos = wgp.Pos
		opts.StdPixels = false
		// fmt.Printf("got prefs for %v: size: %v pos: %v\n", name, opts.Size, opts.Pos)
	}
	win := NewWindow(name, title, opts)
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.HasGeomPrefs = true
	}
	vp := NewViewport2D(width, height)
	vp.SetName("WinVp")
	vp.SetProp("color", &Prefs.FontColor) // everything inherits this..

	win.AddChild(vp)
	win.Viewport = vp
	win.ConfigVLay()
	return win
}

// NewDialogWin creates a new dialog window with given internal handle name,
// display name, and sizing (assumed to be in raw dots), without setting its
// main viewport -- user should do win.AddChild(vp); win.Viewport = vp to set
// their own viewport.
func NewDialogWin(name, title string, width, height int, modal bool) *Window {
	opts := &oswin.NewWindowOptions{
		Title: title, Size: image.Point{width, height}, StdPixels: false,
	}
	opts.SetDialog()
	if modal {
		opts.SetModal()
	}
	wgp := WinGeomPrefs.Pref(name, nil)
	if wgp != nil {
		opts.Size = wgp.Size
		opts.Pos = wgp.Pos
		opts.StdPixels = false
	}
	win := NewWindow(name, title, opts)
	if win == nil {
		return nil
	}
	if wgp != nil {
		win.HasGeomPrefs = true
	}
	return win
}

// ConfigVLay creates and configures the vertical layout as first child of
// Viewport, and installs MainMenu as first element of layout.
func (w *Window) ConfigVLay() {
	vp := w.Viewport
	updt := vp.UpdateStart()
	defer vp.UpdateEnd(updt)
	if !vp.HasChildren() {
		vp.AddNewChild(KiT_Layout, "main-vlay")
	}
	w.MasterVLay = vp.KnownChild(0).(*Layout)
	if !w.MasterVLay.HasChildren() {
		w.MasterVLay.AddNewChild(KiT_MenuBar, "main-menu")
	}
	w.MasterVLay.Lay = LayoutVert
	w.MainMenu = w.MasterVLay.KnownChild(0).(*MenuBar)
	w.MainMenu.SetStretchMaxWidth()
}

// SetMainWidget sets given widget as the main widget for the window -- adds
// into MasterVLay after main menu -- if a main widget has already been set then
// it is deleted and this one replaces it.  Use this method to ensure future
// compatibility.
func (w *Window) SetMainWidget(mw ki.Ki) {
	if len(w.MasterVLay.Kids) == 1 {
		w.MasterVLay.AddChild(mw)
		return
	}
	cmw := w.MasterVLay.KnownChild(1)
	if cmw != mw {
		w.MasterVLay.DeleteChildAtIndex(1, true)
		w.MasterVLay.InsertChild(mw, 1)
	}
}

// SetMainWidgetType sets the main widget of this window to given type
// (typically a Layout or Frame), and returns it.  Adds into MasterVLay after
// main menu -- if a main widget has already been set then it is deleted and
// this one replaces it.  Use this method to ensure future compatibility.
func (w *Window) SetMainWidgetType(typ reflect.Type, name string) ki.Ki {
	if len(w.MasterVLay.Kids) == 1 {
		return w.MasterVLay.AddNewChild(typ, name)
	}
	cmw := w.MasterVLay.KnownChild(1)
	if cmw.Type() != typ {
		w.MasterVLay.DeleteChildAtIndex(1, true)
		return w.MasterVLay.InsertNewChild(typ, 1, name)
	}
	return cmw
}

// SetMainFrame sets the main widget of this window as a Frame, with a default
// column-wise vertical layout and max stretch sizing, and returns that frame.
func (w *Window) SetMainFrame() *Frame {
	fr := w.SetMainWidgetType(KiT_Frame, "main-frame").(*Frame)
	fr.Lay = LayoutVert
	fr.SetStretchMaxWidth()
	fr.SetStretchMaxHeight()
	return fr
}

// SetMainLayout sets the main widget of this window as a Layout, with a default
// column-wise vertical layout and max stretch sizing, and returns it.
func (w *Window) SetMainLayout() *Layout {
	fr := w.SetMainWidgetType(KiT_Layout, "main-lay").(*Layout)
	fr.Lay = LayoutVert
	fr.SetStretchMaxWidth()
	fr.SetStretchMaxHeight()
	return fr
}

// MainWidget returns the main widget for this window -- 2nd element in
// MasterVLay -- returns false if not yet set
func (w *Window) MainWidget() (ki.Ki, bool) {
	return w.MasterVLay.Child(1)
}

// StartEventLoop is the main startup method to call after the initial window
// configuration is setup -- does any necessary final initialization and then
// starts the event loop in this same goroutine, and does not return until the
// window is closed -- see GoStartEventLoop for a version that starts in a
// separate goroutine and returns immediately
func (w *Window) StartEventLoop() {
	w.EventLoop()
}

// GoStartEventLoop starts the event processing loop for this window in a new
// goroutine, and returns immediately
func (w *Window) GoStartEventLoop() {
	go w.EventLoop()
}

// StopEventLoop tells the event loop to stop running when the next event arrives
func (w *Window) StopEventLoop() {
	w.stopEventLoop = true
}

// Quit exits out of the program, closing the window..
func (w *Window) Quit() {
	w.OSWin.Release()
	os.Exit(0) // todo: should use lifecycle
}

// Init performs overall initialization of the gogi system: loading prefs, etc
// -- automatically called when new window opened, but can be called before
// then if pref info needed
func Init() {
	if Prefs.LogicalDPIScale == 0 {
		Prefs.Defaults()
		Prefs.Load()
		Prefs.Apply()
		WinGeomPrefs.Load()
	}
}

// LogicalDPI returns the current logical dots-per-inch resolution of the
// window, which should be used for most conversion of standard units --
// physical DPI can be found in the Screen
func (w *Window) LogicalDPI() float32 {
	if w.OSWin == nil {
		return 96.0 // null default
	}
	return w.OSWin.LogicalDPI()
}

// WinViewport2D returns the viewport directly under this window that serves
// as the master viewport for the entire window
func (w *Window) WinViewport2D() *Viewport2D {
	vpi, ok := w.Children().ElemByType(KiT_Viewport2D, true, 0)
	if !ok { // shouldn't happen
		return nil
	}
	vp, _ := vpi.EmbeddedStruct(KiT_Viewport2D).(*Viewport2D)
	return vp
}

// SetSize requests that the window be resized to the given size -- it will
// trigger a resize event and be processed that way when it occurs
func (w *Window) SetSize(sz image.Point) {
	w.OSWin.SetSize(sz)
}

// Resized updates internal buffers after a window has been resized
func (w *Window) Resized(sz image.Point) {
	if w.IsInactive() || w.Viewport == nil {
		return
	}
	if w.WinTex != nil {
		w.WinTex.Release()
	}
	if w.OverTex != nil {
		w.OverTex.Release()
	}
	w.WinTex, _ = oswin.TheApp.NewTexture(w.OSWin, sz)
	w.OverTex = nil // dynamically allocated when needed
	w.Viewport.Resize(sz)
	WinGeomPrefs.RecordPref(w)
}

// Closed frees any resources after the window has been closed
func (w *Window) Closed() {
	if w.IsInactive() || w.Viewport == nil {
		return
	}
	if w.WinTex != nil {
		w.WinTex.Release()
		w.WinTex = nil
	}
	if w.OverTex != nil {
		w.OverTex.Release()
		w.OverTex = nil
	}
}

// FullReRender performs a full re-render of the window -- each node renders
// into its viewport, aggregating into the main window viewport, which will
// drive an UploadAllViewports call after all the rendering is done, and
// signal the publishing of the window after that
func (w *Window) FullReRender() {
	if w.IsInactive() || w.Viewport == nil {
		return
	}
	pdpi := w.OSWin.PhysicalDPI()
	dpi := oswin.LogicalFmPhysicalDPI(pdpi)
	w.OSWin.SetLogicalDPI(dpi)
	w.Viewport.FullRender2DTree()
	if w.Focus == nil {
		w.SetNextFocusItem()
	}
}

// UploadVpRegion uploads image for one viewport region on the screen, using
// vpBBox bounding box for the viewport, and winBBox bounding box for the
// window -- called after re-rendering specific nodes to update only the
// relevant part of the overall viewport image
func (w *Window) UploadVpRegion(vp *Viewport2D, vpBBox, winBBox image.Rectangle) {
	if w.IsInactive() {
		return
	}
	pr := prof.Start("win.UploadVpRegion")
	if Render2DTrace {
		fmt.Printf("Window: %v uploading region Vp %v, vpbbox: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), vpBBox, w.WinTex.Bounds())
	}
	w.WinTex.Upload(winBBox.Min, vp.OSImage, vpBBox)
	pr.End()
}

// UploadVp uploads entire viewport image for given viewport -- e.g., for
// popups etc updating separately
func (w *Window) UploadVp(vp *Viewport2D, offset image.Point) {
	if w.IsInactive() {
		return
	}
	pr := prof.Start("win.UploadVp")
	if Render2DTrace {
		fmt.Printf("Window: %v uploading Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), vp.OSImage.Bounds(), w.WinTex.Bounds())
	}
	w.WinTex.Upload(offset, vp.OSImage, vp.OSImage.Bounds())
	pr.End()
}

// UploadAllViewports does a complete upload of all active viewports, in the
// proper order, so as to completely refresh the window texture based on
// everything rendered
func (w *Window) UploadAllViewports() {
	if w.IsInactive() {
		return
	}
	pr := prof.Start("win.UploadAllViewports")
	updt := w.UpdateStart()
	if Render2DTrace {
		fmt.Printf("Window: %v uploading full Vp, image bound: %v, wintex bounds: %v\n", w.PathUnique(), w.Viewport.OSImage.Bounds(), w.WinTex.Bounds())
	}
	w.WinTex.Upload(image.ZP, w.Viewport.OSImage, w.Viewport.OSImage.Bounds())
	// then all the current popups
	if w.PopupStack != nil {
		for _, pop := range w.PopupStack {
			gii, _ := KiToNode2D(pop)
			if gii != nil {
				vp := gii.AsViewport2D()
				r := vp.Geom.Bounds()
				if Render2DTrace {
					fmt.Printf("Window: %v uploading popup stack Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), r.Min, vp.OSImage.Bounds())
				}
				w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
			}
		}
	}
	if w.Popup != nil {
		gii, _ := KiToNode2D(w.Popup)
		if gii != nil {
			vp := gii.AsViewport2D()
			r := vp.Geom.Bounds()
			if Render2DTrace {
				fmt.Printf("Window: %v uploading top popup Vp %v, image bound: %v, wintex bounds: %v\n", w.PathUnique(), vp.PathUnique(), r.Min, vp.OSImage.Bounds())
			}
			w.WinTex.Upload(r.Min, vp.OSImage, vp.OSImage.Bounds())
		}
	}
	pr.End()
	w.UpdateEnd(updt) // drives the publish
}

// RenderOverlays -- clears overlay viewport to transparent, renders all
// overlays, uploads result to OverTex
func (w *Window) RenderOverlays() {
	if !w.OverlayVp.HasChildren() {
		w.OverTexActive = false
		return
	}
	updt := w.UpdateStart()
	wsz := w.WinTex.Bounds().Size()
	if w.OverTex == nil || w.OverTex.Bounds() != w.WinTex.Bounds() {
		if w.OverTex != nil {
			w.OverTex.Release()
		}
		w.OverTex, _ = oswin.TheApp.NewTexture(w.OSWin, wsz)
	}
	w.OverlayVp.Win = w
	w.OverlayVp.RenderOverlays(wsz) // handles any resizing etc
	w.OverTex.Upload(image.ZP, w.OverlayVp.OSImage, w.OverlayVp.OSImage.Bounds())
	w.OverTexActive = true
	w.UpdateEnd(updt) // drives the publish
}

// Publish does the final step of updating of the window based on the current
// texture (and overlay texture if active)
func (w *Window) Publish() {
	if w.IsInactive() {
		return
	}
	// fmt.Printf("Win %v doing publish\n", w.Nm)
	pr := prof.Start("win.Publish.Copy")
	w.OSWin.Copy(image.ZP, w.WinTex, w.WinTex.Bounds(), oswin.Src, nil)
	if w.OverTex != nil && w.OverTexActive {
		w.OSWin.Copy(image.ZP, w.OverTex, w.OverTex.Bounds(), oswin.Over, nil)
	}
	pr.End()
	pr2 := prof.Start("win.Publish.Publish")
	w.OSWin.Publish()
	pr2.End()
}

// SignalWindowPublish is the signal receiver function that publishes the
// window updates when the window update signal (UpdateEnd) occurs
func SignalWindowPublish(winki, node ki.Ki, sig int64, data interface{}) {
	win := winki.EmbeddedStruct(KiT_Window).(*Window)
	if Render2DTrace {
		fmt.Printf("Window: %v publishing image due to signal: %v from node: %v\n", win.PathUnique(), ki.NodeSignals(sig), node.PathUnique())
	}
	win.Publish()
}

// Zoom -- positive steps increase logical DPI, negative steps decrease it
func (w *Window) ZoomDPI(steps int) {
	pdpi := w.OSWin.PhysicalDPI()
	dpi := oswin.LogicalFmPhysicalDPI(pdpi)
	dpi += float32(6 * steps)
	oswin.LogicalDPIScale = dpi / pdpi
	w.OSWin.SetLogicalDPI(dpi) // will also be updated by resize events
	fmt.Printf("LogicalDPI now: %v  PhysicalDPI: %v  Scale: %v\n", dpi, pdpi, oswin.LogicalDPIScale)
	w.FullReRender()
}

// ConnectEventType adds a Signal connection for given event type and
// prioritiy to given receiver
func (w *Window) ConnectEventType(recv ki.Ki, et oswin.EventType, pri EventPris, fun ki.RecvFunc) {
	if et >= oswin.EventTypeN {
		log.Printf("Window ConnectEventType type: %v is not a known event type\n", et)
		return
	}
	w.EventSigs[et][pri].Connect(recv, fun)
}

// DisconnectEventType removes Signal connection for given event type to given
// receiver -- pri is priority -- pass AllPris for all priorities
func (w *Window) DisconnectEventType(recv ki.Ki, et oswin.EventType, pri EventPris) {
	if et >= oswin.EventTypeN {
		log.Printf("Window DisconnectEventType type: %v is not a known event type\n", et)
		return
	}
	if pri == AllPris {
		for p := HiPri; p < EventPrisN; p++ {
			w.EventSigs[et][p].Disconnect(recv)
		}
	} else {
		w.EventSigs[et][pri].Disconnect(recv)
	}
}

// DisconnectAllEvents disconnect node from all event signals -- pri is
// priority -- pass AllPris for all priorities
func (w *Window) DisconnectAllEvents(recv ki.Ki, pri EventPris) {
	if pri == AllPris {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			for p := HiPri; p < EventPrisN; p++ {
				w.EventSigs[et][p].Disconnect(recv)
			}
		}
	} else {
		for et := oswin.EventType(0); et < oswin.EventTypeN; et++ {
			w.EventSigs[et][pri].Disconnect(recv)
		}
	}
}

// IsInScope returns true if the given object is in scope for receiving events
func (w *Window) IsInScope(gii Node2D, gi *Node2DBase) bool {
	if w.Popup == nil {
		return true
	}
	if gi.This == w.Popup {
		return true
	}
	if gi.Viewport == nil {
		return false
	}
	if gi.Viewport.This == w.Popup {
		return true
	}
	return false
}

// SendEventSignal sends given event signal to all receivers that want it --
// note that because there is a different EventSig for each event type, we are
// ONLY looking at nodes that have registered to receive that type of event --
// the further filtering is just to ensure that they are in the right position
// to receive the event (focus, popup filtering, etc).
func (w *Window) SendEventSignal(evi oswin.Event) {
	et := evi.Type()
	if et > oswin.EventTypeN || et < 0 {
		return // can't handle other types of events here due to EventSigs[et] size
	}

	// fmt.Printf("got event type: %v\n", et)
	for pri := HiPri; pri < EventPrisN; pri++ {
		if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
			continue
		}
		w.EventSigs[et][pri].EmitFiltered(w.This, int64(et), evi, func(k ki.Ki) bool {
			if k.IsDeleted() { // destroyed is filtered upstream
				return false
			}
			if pri != LowRawPri && evi.IsProcessed() { // someone took care of it
				return false
			}
			gii, gi := KiToNode2D(k)
			if gi != nil {
				if !w.IsInScope(gii, gi) {
					return false
				}
				if evi.OnFocus() && !gii.HasFocus2D() {
					return false
				} else if evi.HasPos() {
					pos := evi.Pos()
					// drag events start with node but can go beyond it..
					_, ok := evi.(*mouse.DragEvent)
					if ok {
						if w.Dragging == gi.This {
							return true
						} else if w.Dragging != nil {
							return false
						} else {
							if pos.In(gi.WinBBox) {
								w.Dragging = gi.This
								bitflag.Set(&gi.Flag, int(NodeDragging))
								return true
							}
							return false
						}
					} else {
						if w.Dragging == gi.This {
							_, dg := KiToNode2D(w.Dragging)
							if dg != nil {
								bitflag.Clear(&dg.Flag, int(NodeDragging))
							}
							w.Dragging = nil
							return true
						}
						if !pos.In(gi.WinBBox) {
							return false
						}
					}
				}
			} else {
				// todo: get a 3D
				return false
			}
			return true
		})
	}
}

// GenMouseFocusEvents processes mouse.MoveEvent to generate mouse.FocusEvent
// events -- returns true if any such events were sent.
func (w *Window) GenMouseFocusEvents(mev *mouse.MoveEvent) bool {
	fe := mouse.FocusEvent{Event: mev.Event}
	pos := mev.Pos()
	ftyp := oswin.MouseFocusEvent
	updated := false
	updt := false
	for pri := HiPri; pri < EventPrisN; pri++ {
		w.EventSigs[ftyp][pri].EmitFiltered(w.This, int64(ftyp), &fe, func(k ki.Ki) bool {
			if k.IsDeleted() { // destroyed is filtered upstream
				return false
			}
			gii, gi := KiToNode2D(k)
			if gi != nil {
				if !w.IsInScope(gii, gi) { // no
					return false
				}
				in := pos.In(gi.WinBBox)
				if in {
					if !bitflag.Has(gi.Flag, int(MouseHasEntered)) {
						fe.Action = mouse.Enter
						bitflag.Set(&gi.Flag, int(MouseHasEntered))
						if !updated {
							updt = w.UpdateStart()
							updated = true
						}
						return true // send event
					} else {
						return false // already in
					}
				} else { // mouse not in object
					if bitflag.Has(gi.Flag, int(MouseHasEntered)) {
						fe.Action = mouse.Exit
						bitflag.Clear(&gi.Flag, int(MouseHasEntered))
						if !updated {
							updt = w.UpdateStart()
							updated = true
						}
						return true // send event
					} else {
						return false // already out
					}
				}
			} else {
				// todo: 3D
				return false
			}
		})
	}
	if updated {
		w.UpdateEnd(updt)
	}
	return updated
}

// SendHoverEvent sends mouse hover event, based on last mouse move event
func (w *Window) SendHoverEvent(e *mouse.MoveEvent) {
	he := mouse.HoverEvent{Event: e.Event}
	he.Processed = false
	he.Action = mouse.Hover
	w.SendEventSignal(&he)
}

// SendKeyChordEvent sends a KeyChord event with given values
func (w *Window) SendKeyChordEvent(r rune, mods ...key.Modifiers) {
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.SetModifiers(mods...)
	ke.Rune = r
	ke.Action = key.Press
	w.SendEventSignal(&ke)
}

// SendKeyFunEvent sends a KeyChord event with params from the given KeyFun
func (w *Window) SendKeyFunEvent(kf KeyFunctions) {
	// todo: do it
}

// PopupIsMenu returns true if the given popup item is a menu
func PopupIsMenu(pop ki.Ki) bool {
	gii, gi := KiToNode2D(pop)
	if gi == nil {
		return false
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsMenu() {
		return true
	}
	return false
}

// PopupIsTooltip returns true if the given popup item is a menu
func PopupIsTooltip(pop ki.Ki) bool {
	gii, gi := KiToNode2D(pop)
	if gi == nil {
		return false
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsTooltip() {
		return true
	}
	return false
}

// PopupIsCompleter returns true if the given popup item is a menu and a completer
func PopupIsCompleter(pop ki.Ki) bool {
	if !PopupIsMenu(pop) {
		return false
	}
	gii, gi := KiToNode2D(pop)
	if gi == nil {
		return false
	}
	vp := gii.AsViewport2D()
	if vp == nil {
		return false
	}
	if vp.IsCompleter() {
		return true
	}
	return false
}

// DeletePopupMenu returns true if the given popup item should be deleted
func (w *Window) DeletePopupMenu(pop ki.Ki, me *mouse.Event) bool {
	if !PopupIsMenu(pop) {
		return false
	}
	if w.NextPopup != nil && PopupIsMenu(w.NextPopup) { // poping up another menu
		return false
	}
	if me.Button != mouse.Left && w.Dragging == nil { // probably menu activation in first place
		return false
	}
	return true
}

// EventLoop runs the event processing loop for the Window -- grabs oswin
// events for the window and dispatches them to receiving nodes, and manages
// other state etc (popups, etc)
func (w *Window) EventLoop() {
	var skippedResize *window.Event

	lastEt := oswin.EventTypeN
	var skipDelta image.Point
	lastSkipped := false

	var startDrag *mouse.DragEvent
	dragStarted := false

	var startDND *mouse.DragEvent
	dndStarted := false

	var startHover, curHover *mouse.MoveEvent
	hoverStarted := false
	var hoverTimer *time.Timer

	for {
		evi := w.OSWin.NextEvent()
		if w.stopEventLoop {
			w.stopEventLoop = false
			fmt.Println("stop event loop")
			break
		}
		if w.DoFullRender {
			// fmt.Printf("Doing full render\n")
			w.DoFullRender = false
			w.FullReRender()
		}
		delPop := false // if true, delete this popup after event loop

		et := evi.Type()
		if et > oswin.EventTypeN || et < 0 { // we don't handle other types of events here
			continue
		}

		now := time.Now()
		lag := now.Sub(evi.Time())
		lagMs := int(lag / time.Millisecond)
		// fmt.Printf("et %v lag %v\n", et, lag)

		if et == lastEt || lastEt == oswin.WindowResizeEvent && et == oswin.PaintEvent {
			switch et {
			case oswin.MouseScrollEvent:
				me := evi.(*mouse.ScrollEvent)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					if !lastSkipped {
						skipDelta = me.Delta
					} else {
						skipDelta = skipDelta.Add(me.Delta)
					}
					lastSkipped = true
					continue
				} else {
					if lastSkipped {
						me.Delta = skipDelta.Add(me.Delta)
					}
					lastSkipped = false
				}
			case oswin.MouseDragEvent:
				me := evi.(*mouse.DragEvent)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					if !lastSkipped {
						skipDelta = me.From
					}
					lastSkipped = true
					continue
				} else {
					if lastSkipped {
						me.From = skipDelta
					}
					lastSkipped = false
				}
			case oswin.WindowResizeEvent:
				we := evi.(*window.Event)
				// fmt.Printf("resize %v\n", we.Size)
				if lagMs > EventSkipLagMSec {
					// fmt.Printf("skipped et %v lag %v\n", et, lag)
					lastSkipped = true
					skippedResize = we
					continue
				} else {
					w.Resized(we.Size)
					w.DoFullRender = true
					lastSkipped = false
					skippedResize = nil
					continue
				}
			case oswin.WindowEvent:
				we := evi.(*window.Event)
				if we.Action == window.Move {
					// fmt.Printf("window %v moved: pos %v winpos: %v\n", w.Nm, we.Pos(), w.OSWin.Position())
					WinGeomPrefs.RecordPref(w)
				}
			case oswin.PaintEvent:
				// fmt.Printf("skipped paint\n")
				continue
			}
		}
		lastSkipped = false
		lastEt = et

		if skippedResize != nil {
			w.Resized(skippedResize.Size)
			w.DoFullRender = true
			skippedResize = nil
		}

		// detect start of drag and DND -- both require delays in starting due
		// to minor wiggles when pressing the mouse button
		if et == oswin.MouseDragEvent {
			if !dragStarted {
				if startDrag == nil {
					startDrag = evi.(*mouse.DragEvent)
				} else {
					delayMs := int(now.Sub(startDrag.Time()) / time.Millisecond)
					if delayMs >= DragStartMSec {
						dst := int(math32.Hypot(float32(startDrag.Where.X-evi.Pos().X), float32(startDrag.Where.Y-evi.Pos().Y)))
						if dst >= DragStartPix {
							dragStarted = true
							startDrag = nil
						}
					}
				}
			}
			if !dndStarted {
				if startDND == nil {
					startDND = evi.(*mouse.DragEvent)
				} else {
					delayMs := int(now.Sub(startDND.Time()) / time.Millisecond)
					if delayMs >= DNDStartMSec {
						dst := int(math32.Hypot(float32(startDND.Where.X-evi.Pos().X), float32(startDND.Where.Y-evi.Pos().Y)))
						if dst >= DNDStartPix {
							dndStarted = true
							w.DNDStartEvent(startDND)
							startDND = nil
						}
					}
				}
			}
		} else {
			dragStarted = false
			startDrag = nil
			dndStarted = false
			startDND = nil
		}

		// detect hover event
		if et == oswin.MouseMoveEvent {
			if !hoverStarted {
				hoverStarted = true
				startHover = evi.(*mouse.MoveEvent)
				curHover = startHover
				hoverTimer = time.AfterFunc(time.Duration(HoverStartMSec)*time.Millisecond, func() {
					w.SendHoverEvent(curHover)
					hoverStarted = false
					startHover = nil
					curHover = nil
					hoverTimer = nil
				})
			} else {
				dst := int(math32.Hypot(float32(startHover.Where.X-evi.Pos().X), float32(startHover.Where.Y-evi.Pos().Y)))
				if dst > HoverMaxPix {
					hoverStarted = false
					startHover = nil
					hoverTimer.Stop()
					hoverTimer = nil
					if w.Popup != nil && PopupIsTooltip(w.Popup) {
						delPop = true
					}
				} else {
					curHover = evi.(*mouse.MoveEvent)
				}
			}
		} else {
			hoverStarted = false
			startHover = nil
			curHover = nil
			if hoverTimer != nil {
				hoverTimer.Stop()
				hoverTimer = nil
			}
		}

		// Window gets first crack at the events, and handles window-specific ones
		switch e := evi.(type) {
		case *lifecycle.Event:
			if e.To == lifecycle.StageDead {
				// fmt.Println("close")
				evi.SetProcessed()
				w.Closed()
				break
			} else {
				// fmt.Printf("lifecycle from: %v to %v\n", e.From, e.To)
				// if e.Crosses(lifecycle.StageFocused) == lifecycle.CrossOff {
				// }
				evi.SetProcessed()
			}
		case *paint.Event:
			w.FullReRender()
			// fmt.Println("doing paint")
			continue
		case *window.Event:
			if e.Action == window.Open || e.Action == window.Resize {
				// fmt.Printf("doing resize for action %v \n", e.Action)
				w.Resized(e.Size)
				w.FullReRender()
			}
			continue
		case *key.ChordEvent:
			kf := KeyFun(e.ChordString())
			w.LastSelMode = mouse.SelectModeMod(e.Modifiers)
			if !e.IsProcessed() {
				if w.Popup != nil && PopupIsMenu(w.Popup) {
					switch kf {
					case KeyFunMoveUp:
						w.SetPrevFocusItem()
						e.SetProcessed()
					case KeyFunMoveDown:
						w.SetNextFocusItem()
						e.SetProcessed()
					}
				}
			}

			if !e.IsProcessed() {
				switch kf {
				case KeyFunFocusNext:
					w.SetNextFocusItem()
					e.SetProcessed()
				case KeyFunFocusPrev:
					w.SetPrevFocusItem()
					e.SetProcessed()
				case KeyFunAbort:
					if w.Popup != nil {
						if PopupIsMenu(w.Popup) || PopupIsTooltip(w.Popup) {
							delPop = true
							e.SetProcessed()
						}
					}
				case KeyFunAccept:
					if w.Popup != nil {
						if PopupIsMenu(w.Popup) || PopupIsTooltip(w.Popup) {
							delPop = true
						}
					}
				case KeyFunGoGiEditor:
					TheViewIFace.GoGiEditor(w.Viewport.This)
					e.SetProcessed()
				case KeyFunZoomIn:
					w.ZoomDPI(1)
					e.SetProcessed()
				case KeyFunZoomOut:
					w.ZoomDPI(-1)
					e.SetProcessed()
				case KeyFunPrefs:
					TheViewIFace.PrefsEditor(&Prefs)
					e.SetProcessed()
				case KeyFunRefresh:
					w.FullReRender()
					// w.UploadAllViewports()
					e.SetProcessed()
				}
			}

			if !e.IsProcessed() {
				cs := e.ChordString()
				switch cs { // some other random special codes, during dev..
				case "Control+Alt+R":
					if prof.Profiling {
						w.EndTargProfile()
						w.EndCPUMemProfile()
					} else {
						w.StartTargProfile()
						w.StartCPUMemProfile()
					}
					e.SetProcessed()
				case "Control+Alt+F":
					w.BenchmarkFullRender()
					e.SetProcessed()
				case "Control+Alt+G":
					w.BenchmarkReRender()
					e.SetProcessed()
				}
			}
			// fmt.Printf("key chord: rune: %v Chord: %v\n", e.Rune, e.ChordString())
		case *mouse.DragEvent:
			w.LastSelMode = mouse.SelectModeMod(e.Modifiers)
			if w.DNDData != nil {
				w.DNDMoveEvent(e)
			} else {
				if !dragStarted {
					e.SetProcessed() // ignore
				}
			}
		case *mouse.Event:
			w.LastSelMode = mouse.SelectModeMod(e.Modifiers)
			if w.DNDData != nil && e.Action == mouse.Release {
				w.DNDDropEvent(e)
			}
		}

		if !evi.IsProcessed() {
			w.SendEventSignal(evi)
			if !delPop && et == oswin.MouseMoveEvent {
				didFocus := w.GenMouseFocusEvents(evi.(*mouse.MoveEvent))
				if didFocus && w.Popup != nil && PopupIsTooltip(w.Popup) {
					delPop = true
				}
			}
		}

		if w.Popup != nil && !delPop {
			if PopupIsTooltip(w.Popup) {
				if et != oswin.MouseMoveEvent {
					delPop = true
				}
			} else if me, ok := evi.(*mouse.Event); ok {
				if me.Action == mouse.Release {
					if w.DeletePopupMenu(w.Popup, me) {
						delPop = true
					}
				}
			}

			if PopupIsCompleter(w.Popup) {
				if et == oswin.KeyChordEvent {
					for pri := HiPri; pri < EventPrisN; pri++ {
						if len(w.FocusStack) > 0 {
							w.EventSigs[et][pri].SendSig(w.FocusStack[0], w.Popup, int64(et), evi)
						}
					}
				}
			}
		}

		if delPop {
			w.PopPopup(w.Popup)
		}

		if w.NextPopup != nil {
			w.PushPopup(w.NextPopup)
			w.NextPopup = nil
		}
	}
	fmt.Println("end of events")
}

// ClearNonFocus clears the focus of any non-w.Focus item -- sometimes can get
// off
func (w *Window) ClearNonFocus() {
	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	updated := false
	updt := false

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		if k == focRoot { // skip top-level
			return true
		}
		// todo: see about 3D guys
		gii, gi := KiToNode2D(k)
		if gi == nil {
			return true
		}
		if w.Focus == k {
			return true
		}
		if gi.HasFocus() {
			if !updated {
				updated = true
				updt = w.UpdateStart()
			}
			bitflag.Clear(&gi.Flag, int(HasFocus))
			gii.FocusChanged2D(false)
		}
		return true
	})
	if updated {
		w.UpdateEnd(updt)
	}
}

// set focus to given item -- returns true if focus changed
func (w *Window) SetFocusItem(k ki.Ki) bool {
	if w.Focus == k {
		return false
	}
	updt := w.UpdateStart()
	if w.Focus != nil {
		gii, gi := KiToNode2D(w.Focus)
		if gi != nil {
			bitflag.Clear(&gi.Flag, int(HasFocus))
			gii.FocusChanged2D(false)
		}
	}
	w.Focus = k
	if k == nil {
		w.UpdateEnd(updt)
		return true
	}
	gii, gi := KiToNode2D(k)
	if gi != nil {
		bitflag.Set(&gi.Flag, int(HasFocus))
		gii.FocusChanged2D(true)
	}
	w.ClearNonFocus()
	w.UpdateEnd(updt)
	return true
}

// set the focus on the next item that can accept focus -- returns true if a focus item found
func (w *Window) SetNextFocusItem() bool {
	gotFocus := false
	focusNext := false // get the next guy
	if w.Focus == nil {
		focusNext = true
	}

	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	for i := 0; i < 2; i++ {
		focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
			if gotFocus {
				return false
			}
			// todo: see about 3D guys
			_, gi := KiToNode2D(k)
			if gi == nil {
				return true
			}
			if w.Focus == k { // current focus can be a non-can-focus item
				focusNext = true
				return true
			}
			if !gi.CanFocus() || gi.VpBBox.Empty() {
				return true
			}
			if focusNext {
				w.SetFocusItem(k)
				gotFocus = true
				return false // done
			}
			return true
		})
		if gotFocus {
			return true
		}
		focusNext = true // this time around, just get the first one
	}
	return false
}

// set the focus on the previous item
func (w *Window) SetPrevFocusItem() bool {
	if w.Focus == nil { // must have a current item here
		w.SetNextFocusItem()
		return false
	}

	gotFocus := false
	var prevItem ki.Ki

	focRoot := w.Viewport.This
	if w.Popup != nil {
		focRoot = w.Popup
	}

	focRoot.FuncDownMeFirst(0, w, func(k ki.Ki, level int, d interface{}) bool {
		if gotFocus {
			return false
		}
		// todo: see about 3D guys
		_, gi := KiToNode2D(k)
		if gi == nil {
			return true
		}
		if w.Focus == k {
			gotFocus = true
			return false
		}
		if !gi.CanFocus() || gi.VpBBox.Empty() {
			return true
		}
		prevItem = k
		return true
	})
	if gotFocus && prevItem != nil {
		w.SetFocusItem(prevItem)
	} else {
		w.SetNextFocusItem()
	}
	return true
}

// push current popup onto stack and set new popup
func (w *Window) PushPopup(pop ki.Ki) {
	if w.PopupStack == nil {
		w.PopupStack = make([]ki.Ki, 0, 50)
	}
	pop.SetParent(w.This) // popup has parent as window -- draws directly in to assoc vp
	w.PopupStack = append(w.PopupStack, w.Popup)
	w.Popup = pop
	_, gi := KiToNode2D(pop)
	if gi != nil {
		gi.FullRender2DTree()
	}
	w.PushFocus(pop)
}

// disconnect given popup -- typically the current one
func (w *Window) DisconnectPopup(pop ki.Ki) {
	w.DisconnectAllEvents(pop, AllPris)
	pop.SetParent(nil) // don't redraw the popup anymore
	w.Viewport.UploadToWin()
}

// close given popup -- must be the current one -- returns false if not
func (w *Window) ClosePopup(pop ki.Ki) bool {
	if pop != w.Popup {
		return false
	}
	w.DisconnectPopup(pop)
	w.PopPopup(pop)
	return true
}

// pop current popup off the popup stack and set to current popup
func (w *Window) PopPopup(pop ki.Ki) {
	gii, ok := pop.(Node2D)
	if ok {
		pvp := gii.AsViewport2D()
		if pvp != nil {
			pvp.DeletePopup()
		}
	}
	sz := len(w.PopupStack)
	if w.Popup == pop {
		if w.PopupStack == nil || sz == 0 {
			w.Popup = nil
		} else {
			w.Popup = w.PopupStack[sz-1]
			w.PopupStack = w.PopupStack[:sz-1]
		}
		w.PopFocus()
	} else {
		for i := sz - 1; i >= 0; i-- {
			pp := w.PopupStack[i]
			if pp == pop {
				w.PopupStack = w.PopupStack[:i+copy(w.PopupStack[i:], w.PopupStack[i+1:])]
				break
			}
		}
		// do nothing
	}
	w.UploadAllViewports()
}

// push current focus onto stack and set new focus
func (w *Window) PushFocus(p ki.Ki) {
	if w.FocusStack == nil {
		w.FocusStack = make([]ki.Ki, 0, 50)
	}
	w.FocusStack = append(w.FocusStack, w.Focus)
	w.Focus = p
	w.SetNextFocusItem()
}

// pop Mask off the focus stack and set to current focus
func (w *Window) PopFocus() {
	if w.FocusStack == nil || len(w.FocusStack) == 0 {
		w.Focus = nil
		return
	}
	sz := len(w.FocusStack)
	w.Focus = w.FocusStack[sz-1]
	w.FocusStack = w.FocusStack[:sz-1]
}

///////////////////////////////////////////////////////
// Drag-n-drop

// StartDragNDrop is called by a node to start a drag-n-drop operation on
// given source node, which is responsible for providing the data and image
// representation of the node
func (w *Window) StartDragNDrop(src ki.Ki, data mimedata.Mimes, img Node2D) {
	// todo: 3d version later..
	w.DNDSource = src
	w.DNDData = data
	wimg := img.AsWidget()
	if _, sgi := KiToNode2D(src); sgi != nil { // 2d case
		if sw := sgi.AsWidget(); sw != nil {
			wimg.LayData.AllocPos.SetPoint(sw.LayData.AllocPos.ToPoint())
		}
	}
	wimg.This.SetName(src.UniqueName())
	w.OverlayVp.AddChild(wimg.This)
	w.DNDImage = wimg.This
	// fmt.Printf("starting dnd: %v\n", src.Name())
}

// FinalizeDragNDrop is called by a node to finalize the drag-n-drop
// operation, after given action has been performed on the target -- allows
// target to cancel, by sending dnd.DropIgnore
func (w *Window) FinalizeDragNDrop(action dnd.DropMods) {
	if w.DNDFinalEvent == nil { // shouldn't happen...
		return
	}
	de := w.DNDFinalEvent
	de.Processed = false
	de.Mod = action
	if de.Source != nil {
		et := de.Type()
		de.Action = dnd.DropFmSource
		for pri := HiPri; pri < EventPrisN; pri++ {
			w.EventSigs[et][pri].SendSig(de.Source, w, int64(et), (oswin.Event)(de))
		}
	}
	w.DNDFinalEvent = nil
}

// DNDSetCursor sets the cursor based on the DND event mod
func (w *Window) DNDSetCursor(dmod dnd.DropMods) {
	if w.DNDMod == dmod {
		return
	}
	if w.DNDMod != dnd.NoDropMod {
		oswin.TheApp.Cursor().Pop()
	}
	switch dmod {
	case dnd.DropCopy:
		oswin.TheApp.Cursor().Push(cursor.DragCopy)
	case dnd.DropMove:
		oswin.TheApp.Cursor().Push(cursor.DragMove)
	case dnd.DropLink:
		oswin.TheApp.Cursor().Push(cursor.DragLink)
	}
	w.DNDMod = dmod
}

// DNDClearCursor clears any existing DND cursor that might have been set
func (w *Window) DNDClearCursor() {
	if w.DNDMod == dnd.NoDropMod {
		return
	}
	oswin.TheApp.Cursor().Pop()
	w.DNDMod = dnd.NoDropMod
}

// DNDStartEvent handles drag-n-drop start events
func (w *Window) DNDStartEvent(e *mouse.DragEvent) {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.Processed = false
	de.Action = dnd.Start
	de.DefaultMod()
	w.DNDMod = dnd.NoDropMod
	w.SendEventSignal(&de)
	// now up to receiver to call StartDragNDrop if they want to..
}

// DNDMoveEvent handles drag-n-drop move events
func (w *Window) DNDMoveEvent(e *mouse.DragEvent) {
	if gii, _ := KiToNode2D(w.DNDImage); gii != nil { // 2d case
		if wg := gii.AsWidget(); wg != nil {
			wg.LayData.AllocPos.SetPoint(e.Where)
		}
	} // else 3d..
	// todo: when e.Where goes negative, transition to OS DND
	// todo: send move / enter / exit events to anyone listening
	de := dnd.MoveEvent{Event: dnd.Event{EventBase: e.Event.EventBase, Where: e.Event.Where, Modifiers: e.Event.Modifiers}, From: e.From, LastTime: e.LastTime}
	de.Processed = false
	de.DefaultMod()
	de.Action = dnd.Move
	w.DNDSetCursor(de.Mod)
	w.SendEventSignal(&de)
	w.RenderOverlays()
	e.SetProcessed()
}

// DNDDropEvent handles drag-n-drop drop event (action = release)
func (w *Window) DNDDropEvent(e *mouse.Event) {
	de := dnd.Event{EventBase: e.EventBase, Where: e.Where, Modifiers: e.Modifiers}
	de.Processed = false
	de.DefaultMod()
	de.Action = dnd.DropOnTarget
	de.Data = w.DNDData
	de.Source = w.DNDSource
	bitflag.Clear(w.DNDSource.Flags(), int(NodeDragging))
	w.Dragging = nil
	w.SendEventSignal(&de)
	w.DNDFinalEvent = &de
	w.ClearDragNDrop()
	e.SetProcessed()
}

// ClearDragNDrop clears any existing DND values
func (w *Window) ClearDragNDrop() {
	w.DNDSource = nil
	w.DNDData = nil
	w.OverlayVp.DeleteChild(w.DNDImage, true)
	w.DNDImage = nil
	w.DNDClearCursor()
	w.Dragging = nil
	w.RenderOverlays()
}

///////////////////////////////////////////////////////
// Profiling and Benchmarking, controlled by hot-keys

func (w *Window) StartCPUMemProfile() {
	fmt.Println("Starting Std CPU / Mem Profiling")
	f, err := os.Create("cpu.prof")
	if err != nil {
		log.Fatal("could not create CPU profile: ", err)
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal("could not start CPU profile: ", err)
	}
}

func (w *Window) EndCPUMemProfile() {
	fmt.Println("Ending Std CPU / Mem Profiling")
	pprof.StopCPUProfile()
	f, err := os.Create("mem.prof")
	if err != nil {
		log.Fatal("could not create memory profile: ", err)
	}
	runtime.GC() // get up-to-date statistics
	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Fatal("could not write memory profile: ", err)
	}
	f.Close()
}

// start targeted profiling using prof package
func (w *Window) StartTargProfile() {
	nn := 0
	w.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		nn++
		return true
	})
	fmt.Printf("Starting Targeted Profiling, window has %v nodes\n", nn)
	prof.Reset()
	prof.Profiling = true
}

// end targeted profiling and print report
func (w *Window) EndTargProfile() {
	prof.Report(time.Millisecond)
	prof.Profiling = false
}

// run benchmark of 50 full re-renders, report targeted profile results
func (w *Window) BenchmarkFullRender() {
	fmt.Println("Starting BenchmarkFullRender")
	w.StartCPUMemProfile()
	w.StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.FullRender2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	w.EndTargProfile()
	w.EndCPUMemProfile()
}

// run benchmark of 50 just-re-renders, not full rebuilds
func (w *Window) BenchmarkReRender() {
	fmt.Println("Starting BenchmarkReRender")
	w.StartTargProfile()
	ts := time.Now()
	n := 50
	for i := 0; i < n; i++ {
		w.Viewport.Render2DTree()
	}
	td := time.Now().Sub(ts)
	fmt.Printf("Time for %v Re-Renders: %12.2f s\n", n, float64(td)/float64(time.Second))
	w.EndTargProfile()
}

//////////////////////////////////////////////////////////////////////////////////
//  WindowGeom

var WinGeomPrefs = WindowGeomPrefs{}

// WindowGeom records the geometry settings used for a given window
type WindowGeom struct {
	WinName    string
	Screen     string
	LogicalDPI float32
	Size       image.Point
	Pos        image.Point
}

// WindowGeomPrefs records the window geometry by window name, screen name --
// looks up the info automatically for new windows and saves persistently
type WindowGeomPrefs map[string]map[string]WindowGeom

// WinGeomPrefsFileName is the name of the preferences file in GoGi prefs directory
var WinGeomPrefsFileName = "win_geom_prefs.json"

// Load Window Geom preferences from GoGi standard prefs directory
func (wg *WindowGeomPrefs) Load() error {
	if wg == nil {
		*wg = make(WindowGeomPrefs, 100)
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		log.Println(err)
		return err
	}
	err = json.Unmarshal(b, wg)
	if err != nil {
		log.Println(err)
	}
	return err
}

// Save Window Geom Preferences to GoGi standard prefs directory
func (wg *WindowGeomPrefs) Save() error {
	if wg == nil {
		return nil
	}
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, WinGeomPrefsFileName)
	b, err := json.MarshalIndent(wg, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	return err
}

// RecordPref records current state of window as preference
func (wg *WindowGeomPrefs) RecordPref(win *Window) {
	if wg == nil {
		*wg = make(WindowGeomPrefs, 100)
	}
	sc := win.OSWin.Screen()
	wgr := WindowGeom{WinName: win.Nm, Screen: sc.Name, LogicalDPI: win.LogicalDPI()}
	wgr.Pos = win.OSWin.Position()
	wgr.Size = win.OSWin.Size()
	if wgr.Size == image.ZP {
		// fmt.Printf("Pref: NOT storing null size for win: %v scrn: %v\n", win.Nm, sc.Name)
		return
	}
	if (*wg)[win.Nm] == nil {
		(*wg)[win.Nm] = make(map[string]WindowGeom, 10)
	}
	(*wg)[win.Nm][sc.Name] = wgr
	wg.Save()
}

// Pref returns an existing preference for given window name, or one adapted
// to given screen if only records are on a different screen -- if scrn is nil
// then default (first) screen is used from oswin.TheApp
func (wg *WindowGeomPrefs) Pref(winName string, scrn *oswin.Screen) *WindowGeom {
	if wg == nil {
		return nil
	}
	wps, ok := (*wg)[winName]
	if !ok {
		return nil
	}

	if scrn == nil {
		scrn = oswin.TheApp.Screen(0)
		// fmt.Printf("Pref: using scrn 0: %v\n", scrn.Name)
	}

	wp, ok := wps[scrn.Name]
	if ok {
		return &wp
	}

	if len(wps) == 0 { // shouldn't happen
		return nil
	}

	trgdpi := scrn.LogicalDPI
	fmt.Printf("Pref: falling back on dpi conversion: %v\n", trgdpi)

	// try to find one with same logical dpi, else closest
	var closest *WindowGeom
	minDPId := float32(100000.0)
	for _, wp = range wps {
		if wp.LogicalDPI == trgdpi {
			return &wp
		}
		dpid := math32.Abs(wp.LogicalDPI - trgdpi)
		if dpid < minDPId {
			minDPId = dpid
			closest = &wp
		}
	}

	wp = *closest
	rescale := trgdpi / closest.LogicalDPI
	wp.Pos.X = int(float32(wp.Pos.X) * rescale)
	wp.Pos.Y = int(float32(wp.Pos.Y) * rescale)
	wp.Size.X = int(float32(wp.Size.X) * rescale)
	wp.Size.Y = int(float32(wp.Size.Y) * rescale)
	fmt.Printf("Pref: rescaled pos: %v size: %v\n", wp.Pos, wp.Size)
	return &wp
}

//////////////////////////////////////////////////////////////////////////////
//  ViewIFace

// ViewIFace is an interface into the View GUI types in giv subpackage
type ViewIFace interface {

	// GoGiEditor opens an interactive editor of given Ki tree, at its root
	GoGiEditor(obj ki.Ki)

	// PrefsEditor opens an interactive editor of given preferences object
	PrefsEditor(prefs *Preferences)
}

// TheViewIFace is the implemenation of the interface, defined in giv package
var TheViewIFace ViewIFace
