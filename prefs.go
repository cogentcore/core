// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki"
)

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences
type ScreenPrefs struct {
	LogicalDPIScale float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
}

// Preferences are the overall user preferences for GoGi, providing some basic
// customization -- in addition, most gui settings can be styled using
// CSS-style sheets under CustomStyle.  These prefs are saved and loaded from
// the GoGi user preferences directory -- see oswin/App for further info
type Preferences struct {
	LogicalDPIScale float32                `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
	ScreenPrefs     map[string]ScreenPrefs `desc:"screen-specific preferences -- will override overall defaults if set"`
	DoubleClickMSec int                    `min:"100" step:"50" desc:"the maximum time interval in msec between button press events to count as a double-click"`
	ScrollWheelRate int                    `min:"1" step:"1" desc:"how fast the scroll wheel moves -- typically pixels per wheel step -- only used for OS's that do not have a native preference for this (e.g., X11)"`
	FontColor       Color                  `desc:"default font / pen color"`
	BackgroundColor Color                  `desc:"default background color"`
	ShadowColor     Color                  `desc:"color for shadows -- should generally be a darker shade of the background color"`
	BorderColor     Color                  `desc:"default border color, for button, frame borders, etc"`
	ControlColor    Color                  `desc:"default main color for controls: buttons, etc"`
	IconColor       Color                  `desc:"color for icons or other solidly-colored, small elements"`
	SelectColor     Color                  `desc:"color for selected elements"`
	HighlightColor  Color                  `desc:"color for highlight background"`
	StdKeyMapName   string                 `desc:"name of standard key map -- select via Std KeyMap button in editor"`
	CustomKeyMap    KeyMap                 `desc:"customized mapping from keys to interface functions"`
	PrefsOverride   bool                   `desc:"if true my custom style preferences override other styling -- otherwise they provide defaults that can be overriden by app-specific styling"`
	CustomStyles    ki.Props               `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	FontPaths       []string               `desc:"extra font paths, beyond system defaults -- searched first"`
	FavPaths        FavPaths               `desc:"favorite paths, shown in FileViewer and also editable there"`
}

// Prefs are the overall preferences
var Prefs = Preferences{}

func (p *Preferences) Defaults() {
	p.LogicalDPIScale = oswin.LogicalDPIScale
	p.DoubleClickMSec = 500
	p.ScrollWheelRate = 20
	p.FontColor.SetColor(color.Black)
	p.BorderColor.SetString("#666", nil)
	p.BackgroundColor.SetColor(color.White)
	p.ShadowColor.SetString("darker-10", &p.BackgroundColor)
	p.ControlColor.SetString("#EEF", nil)
	p.IconColor.SetString("highlight-30", p.ControlColor)
	p.SelectColor.SetString("#CFC", nil)
	p.HighlightColor.SetString("#FFA", nil)
	p.FavPaths.SetToDefaults()
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "prefs.json"

// Load preferences from GoGi standard prefs directory
func (p *Preferences) Load() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err) // ok to be non-existant
		return err
	}
	return json.Unmarshal(b, p)
}

// Save Preferences to GoGi standard prefs directory
func (p *Preferences) Save() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
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
	np := len(p.FavPaths)
	for i := 0; i < np; i++ {
		if p.FavPaths[i].Ic == "" {
			p.FavPaths[i].Ic = "folder"
		}
	}

	oswin.LogicalDPIScale = p.LogicalDPIScale
	mouse.DoubleClickMSec = p.DoubleClickMSec
	mouse.ScrollWheelRate = p.ScrollWheelRate
	if p.StdKeyMapName != "" {
		defmap := StdKeyMapByName(p.StdKeyMapName)
		if defmap != nil {
			DefaultKeyMap = defmap
		}
	}
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
		if _, ok := p.ScreenPrefs[sc.Name]; ok {
			// todo: this is not currently used -- need to update code in window.go
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(sc.PhysicalDPI)
		} else {
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(sc.PhysicalDPI)
		}
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

// SetKeyMap installs the given keymap as the current CustomKeyMap, which can
// then be customized
func (p *Preferences) SetKeyMap(kmap *KeyMap) {
	p.CustomKeyMap = make(KeyMap, len(*kmap))
	for key, val := range *kmap {
		p.CustomKeyMap[key] = val
	}
}

// ScreenInfo displays screen info for all screens on the console
func (p *Preferences) ScreenInfo() {
	ns := oswin.TheApp.NScreens()
	for i := 0; i < ns; i++ {
		sc := oswin.TheApp.Screen(i)
		fmt.Printf("Screen number: %v name: %v\n%+v\n", i, sc.Name, sc)
	}
}

// PrefColor returns preference color of given name (case insensitive)
func (p *Preferences) PrefColor(clrName string) *Color {
	lc := strings.Replace(strings.ToLower(clrName), "-", "", -1)
	switch lc {
	case "fontcolor":
		return &p.FontColor
	case "backgroundcolor":
		return &p.BackgroundColor
	case "shadowcolor":
		return &p.ShadowColor
	case "bordercolor":
		return &p.BorderColor
	case "controlcolor":
		return &p.ControlColor
	case "iconcolor":
		return &p.IconColor
	case "selectcolor":
		return &p.SelectColor
	case "highlightcolor":
		return &p.HighlightColor
	}
	log.Printf("Preference color %v (simlified to: %v) not found\n", clrName, lc)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//  FavoritePaths

// FavPathItem represents one item in a favorite path list, for display of
// favorites.  Is an ordered list instead of a map because user can organize
// in order
type FavPathItem struct {
	Ic   IconName `desc:"icon for item"`
	Name string   `width:"20" desc:"name of the favorite item"`
	Path string   `tableview:"-select"`
}

// FavPaths is a list (slice) of favorite path items
type FavPaths []FavPathItem

// SetToDefaults sets the paths to default values
func (p *FavPaths) SetToDefaults() {
	*p = make(FavPaths, len(DefaultPaths))
	copy(*p, DefaultPaths)
}

// DefaultPaths are default favorite paths
var DefaultPaths = FavPaths{
	{"home", "home", "~"},
	{"desktop", "Desktop", "~/Desktop"},
	{"documents", "Documents", "~/Documents"},
	{"folder-download", "Downloads", "~/Downloads"},
	{"computer", "root", "/"},
}
