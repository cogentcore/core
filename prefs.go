// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
)

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences
type ScreenPrefs struct {
	LogicalDPIScale float32 `desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
}

// Preferences are the overall user preferences for GoGi, providing some basic
// customization -- in addition, most gui settings can be styled using
// CSS-style sheets under CustomStyle.  These prefs are saved and loaded from
// the GoGi user preferences directory -- see oswin/App for further info
type Preferences struct {
	LogicalDPIScale  float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
	ScreenPrefs      map[string]Preferences
	DialogsSepWindow bool     `desc:"do dialog windows open in a separate OS-level window, or do they open within the same parent window"`
	DoubleClickMSec  int      `min:"100" step:"50" desc:"the maximum time interval in msec between button press events to count as a double-click"`
	ScrollWheelRate  int      `min:"1" step:"1" desc:"how fast the scroll wheel moves -- typically pixels per wheel step -- only used for OS's that do not have a native preference for this (e.g., X11)"`
	FontColor        Color    `desc:"default font / pen color"`
	BackgroundColor  Color    `desc:"default background color"`
	ShadowColor      Color    `desc:"color for shadows -- should generally be a darker shade of the background color"`
	BorderColor      Color    `desc:"default border color, for button, frame borders, etc"`
	ControlColor     Color    `desc:"default main color for controls: buttons, etc"`
	IconColor        Color    `desc:"color for icons or other solidly-colored, small elements"`
	SelectColor      Color    `desc:"color for selected elements"`
	CustomKeyMap     KeyMap   `desc:"customized mapping from keys to interface functions"`
	PrefsOverride    bool     `desc:"if true my custom style preferences override other styling -- otherwise they provide defaults that can be overriden by app-specific styling"`
	CustomStyles     ki.Props `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	FontPaths        []string `desc:"extra font paths, beyond system defaults -- searched first"`
}

// Prefs are the overall preferences
var Prefs = Preferences{}

func (p *Preferences) Defaults() {
	p.LogicalDPIScale = oswin.LogicalDPIScale
	p.DialogsSepWindow = true
	p.DoubleClickMSec = 500
	p.ScrollWheelRate = 20
	p.FontColor.SetColor(color.Black)
	p.BorderColor.SetString("#666", nil)
	p.BackgroundColor.SetColor(color.White)
	p.ShadowColor.SetString("darker-10", &p.BackgroundColor)
	p.ControlColor.SetString("#EEF", nil)
	p.IconColor.SetString("darker-30", p.ControlColor)
	p.SelectColor.SetString("#CFC", nil)
}

// Load preferences from GoGi standard prefs directory
func (p *Preferences) Load() error {
	pdir := oswin.TheApp.GoGiPrefsDir()

	pnm := filepath.Join(pdir, "prefs.json")
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err)
		return err
	}
	return json.Unmarshal(b, p)
}

// Save Preferences to GoGi standard prefs directory
func (p *Preferences) Save() error {
	pdir := oswin.TheApp.GoGiPrefsDir()

	pnm := filepath.Join(pdir, "prefs.json")

	b, err := json.MarshalIndent(p, "", "  ")
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

// Apply preferences to all the relevant settings
func (p *Preferences) Apply() {
	oswin.LogicalDPIScale = p.LogicalDPIScale
	mouse.DoubleClickMSec = p.DoubleClickMSec
	mouse.ScrollWheelRate = p.ScrollWheelRate
	DialogsSepWindow = p.DialogsSepWindow
	if p.CustomKeyMap != nil {
		SetActiveKeyMap(&Prefs.CustomKeyMap) // fills in missing pieces
	}
	if p.FontPaths != nil {
		paths := append(p.FontPaths, oswin.TheApp.FontPaths()...)
		FontLibrary.InitFontPaths(paths...)
	} else {
		FontLibrary.InitFontPaths(oswin.TheApp.FontPaths()...)
	}
	n := oswin.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := oswin.TheApp.Screen(i)
		sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(sc.PhysicalDPI)
	}
}

// Update everything with current preferences -- triggers rebuild of default styles
func (p *Preferences) Update() {
	p.Apply()

	RebuildDefaultStyles = true
	n := oswin.TheApp.NWindows()
	for i := 0; i < n; i++ {
		owin := oswin.TheApp.Window(i)
		if win, ok := owin.Parent().(*Window); ok {
			win.FullReRender()
		}
	}
	RebuildDefaultStyles = false
	// needs another pass through to get it right..
	for i := 0; i < n; i++ {
		owin := oswin.TheApp.Window(i)
		if win, ok := owin.Parent().(*Window); ok {
			win.FullReRender()
		}
	}
}

// DefaultKeyMap installs the current default key map, prior to editing
func (p *Preferences) DefaultKeyMap() {
	p.CustomKeyMap = make(KeyMap, len(DefaultKeyMap))
	for key, val := range DefaultKeyMap {
		p.CustomKeyMap[key] = val
	}
}

// Edit Preferences in a separate window
func (p *Preferences) Edit() {
	width := 800
	height := 600
	win := NewWindow2D("GoGi Preferences", width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()
	vp.SetProp("background-color", color.White)
	vp.Fill = true

	vlay := vp.AddNewChild(KiT_Frame, "vlay").(*Frame)
	vlay.Lay = LayoutCol

	trow := vlay.AddNewChild(KiT_Layout, "trow").(*Layout)
	trow.Lay = LayoutRow
	trow.SetStretchMaxWidth()

	spc := vlay.AddNewChild(KiT_Space, "spc1").(*Space)
	spc.SetFixedHeight(units.NewValue(2.0, units.Em))

	trow.AddNewChild(KiT_Stretch, "str1")
	title := trow.AddNewChild(KiT_Label, "title").(*Label)
	title.Text = "GoGi Preferences"
	title.SetStretchMaxWidth()
	trow.AddNewChild(KiT_Stretch, "str2")

	sv := vlay.AddNewChild(KiT_StructView, "sv").(*StructView)
	sv.SetStruct(p, nil)
	sv.SetStretchMaxWidth()
	sv.SetStretchMaxHeight()

	bspc := vlay.AddNewChild(KiT_Space, "ButSpc").(*Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	brow := vlay.AddNewChild(KiT_Layout, "brow").(*Layout)
	brow.Lay = LayoutRow
	brow.SetProp("align-horiz", "center")
	brow.SetStretchMaxWidth()

	up := brow.AddNewChild(KiT_Button, "update").(*Button)
	up.SetText("Update")
	up.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			p.Update()
		}
	})

	savej := brow.AddNewChild(KiT_Button, "savejson").(*Button)
	savej.SetText("Save")
	savej.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			p.Save()
		}
	})

	loadj := brow.AddNewChild(KiT_Button, "loadjson").(*Button)
	loadj.SetText("Load")
	loadj.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			p.Load()
		}
	})

	defmap := brow.AddNewChild(KiT_Button, "defkemap").(*Button)
	defmap.SetText("Default KeyMap")
	defmap.ButtonSig.Connect(win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		if sig == int64(ButtonClicked) {
			p.DefaultKeyMap()
		}
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop()
}
