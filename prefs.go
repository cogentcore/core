// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"image/color"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
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
	LogicalDPIScale float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
	ScreenPrefs     map[string]Preferences

	FontColor       Color    `desc:"default font / pen color"`
	BackgroundColor Color    `desc:"default background color"`
	ShadowColor     Color    `desc:"color for shadows -- should generally be a darker shade of the background color"`
	BorderColor     Color    `desc:"default border color, for button, frame borders, etc"`
	ControlColor    Color    `desc:"default main color for controls: buttons, etc"`
	IconColor       Color    `desc:"color for icons or other solidly-colored, small elements"`
	SelectColor     Color    `desc:"color for selected elements"`
	CustomKeyMap    KeyMap   `desc:"customized mapping from keys to interface functions"`
	PrefsOverride   bool     `desc:"if true my custom style preferences override other styling -- otherwise they provide defaults that can be overriden by app-specific styling"`
	CustomStyles    ki.Props `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	FontPaths       []string `desc:"extra font paths, beyond system defaults -- searched first"`
}

// Prefs are the overall preferences
var Prefs = Preferences{}

func (p *Preferences) Defaults() {
	p.LogicalDPIScale = 0.6 // most people have Hi-DPI these days?
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
		log.Println(err)
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

	bspc := vlay.AddNewChild(KiT_Space, "ButSpc").(*Space)
	bspc.SetFixedHeight(units.NewValue(1.0, units.Em))

	brow := vlay.AddNewChild(KiT_Layout, "brow").(*Layout)
	brow.Lay = LayoutRow
	brow.SetProp("align-horiz", "center")
	brow.SetStretchMaxWidth()

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
	go win.StartEventLoopNoWait()
}
