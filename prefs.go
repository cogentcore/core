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

// ZoomFactor is a temporary multiplier on LogicalDPI used for per-session
// display zooming without changing prefs -- see SaveCurrentZoom to save prefs
// with this current factor.
var ZoomFactor = float32(1.0)

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences.
type ScreenPrefs struct {
	LogicalDPIScale float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
}

// ColorPrefs specify colors for all major categories of GUI elements, and are
// used in the default styles.
type ColorPrefs struct {
	Font       Color `desc:"default font / pen color"`
	Background Color `desc:"default background color"`
	Shadow     Color `desc:"color for shadows -- should generally be a darker shade of the background color"`
	Border     Color `desc:"default border color, for button, frame borders, etc"`
	Control    Color `desc:"default main color for controls: buttons, etc"`
	Icon       Color `desc:"color for icons or other solidly-colored, small elements"`
	Select     Color `desc:"color for selected elements"`
	Highlight  Color `desc:"color for highlight background"`
	Link       Color `desc:"color for links in text etc"`
}

// ParamPrefs contains misc parameters controlling GUI behavior.
type ParamPrefs struct {
	DoubleClickMSec int  `min:"100" step:"50" desc:"the maximum time interval in msec between button press events to count as a double-click"`
	ScrollWheelRate int  `min:"1" step:"1" desc:"how fast the scroll wheel moves -- typically pixels per wheel step -- only used for OS's that do not have a native preference for this (e.g., X11)"`
	LocalMainMenu   bool `desc:"controls whether the main menu is displayed locally at top of each window, in addition to global menu at the top of the screen.  Mac native apps do not do this, but OTOH it makes things more consistent with other platforms, and with larger screens, it can be convenient to have access to all the menu items right there."`
}

// Preferences are the overall user preferences for GoGi, providing some basic
// customization -- in addition, most gui settings can be styled using
// CSS-style sheets under CustomStyle.  These prefs are saved and loaded from
// the GoGi user preferences directory -- see oswin/App for further info.
type Preferences struct {
	LogicalDPIScale float32                `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
	ScreenPrefs     map[string]ScreenPrefs `desc:"screen-specific preferences -- will override overall defaults if set"`
	Colors          ColorPrefs             `desc:"color preferences"`
	Params          ParamPrefs             `desc:"parameters controlling GUI behavior"`
	StdKeyMapName   string                 `desc:"name of standard key map -- select via Std KeyMap button in editor"`
	CustomKeyMap    KeyMap                 `desc:"customized mapping from keys to interface functions"`
	PrefsOverride   bool                   `desc:"if true my custom style preferences override other styling -- otherwise they provide defaults that can be overriden by app-specific styling"`
	CustomStyles    ki.Props               `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	FontFamily      FontName               `desc:"default font family when otherwise not specified"`
	FontPaths       []string               `desc:"extra font paths, beyond system defaults -- searched first"`
	FavPaths        FavPaths               `desc:"favorite paths, shown in FileViewer and also editable there"`
	SavedPathsMax   int                    `desc:"maximum number of saved paths to save in FileView"`
	FileViewSort    string                 `desc:"column to sort by in FileView, and :up or :down for direction -- updated automatically via FileView"`
}

// Prefs are the overall preferences
var Prefs = Preferences{}

func (p *ColorPrefs) Defaults() {
	p.Font.SetColor(color.Black)
	p.Border.SetString("#666", nil)
	p.Background.SetColor(color.White)
	p.Shadow.SetString("darker-10", &p.Background)
	p.Control.SetString("#EEF", nil)
	p.Icon.SetString("highlight-30", p.Control)
	p.Select.SetString("#CFC", nil)
	p.Highlight.SetString("#FFA", nil)
	p.Link.SetString("#00F", nil)
}

// PrefColor returns preference color of given name (case insensitive)
func (p *ColorPrefs) PrefColor(clrName string) *Color {
	lc := strings.Replace(strings.ToLower(clrName), "-", "", -1)
	switch lc {
	case "font":
		return &p.Font
	case "background":
		return &p.Background
	case "shadow":
		return &p.Shadow
	case "border":
		return &p.Border
	case "control":
		return &p.Control
	case "icon":
		return &p.Icon
	case "select":
		return &p.Select
	case "highlight":
		return &p.Highlight
	case "link":
		return &p.Link
	}
	log.Printf("Preference color %v (simlified to: %v) not found\n", clrName, lc)
	return nil
}

// todo: Save, Load colors separately!

func (p *ParamPrefs) Defaults() {
	p.DoubleClickMSec = 500
	p.ScrollWheelRate = 20
	p.LocalMainMenu = false
}

func (p *Preferences) Defaults() {
	p.LogicalDPIScale = 1.0
	p.Colors.Defaults()
	p.Params.Defaults()
	p.FavPaths.SetToDefaults()
	p.FontFamily = "Go"
	p.SavedPathsMax = 20
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

// Apply preferences to all the relevant settings.
func (p *Preferences) Apply() {
	np := len(p.FavPaths)
	for i := 0; i < np; i++ {
		if p.FavPaths[i].Ic == "" {
			p.FavPaths[i].Ic = "folder"
		}
	}

	mouse.DoubleClickMSec = p.Params.DoubleClickMSec
	mouse.ScrollWheelRate = p.Params.ScrollWheelRate
	LocalMainMenu = p.Params.LocalMainMenu

	if p.StdKeyMapName != "" {
		defmap, _ := StdKeyMapByName(p.StdKeyMapName)
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
	p.ApplyDPI()
}

// ApplyDPI updates the screen LogicalDPI values according to current
// preferences and zoom factor, and then updates all open windows as well.
func (p *Preferences) ApplyDPI() {
	n := oswin.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := oswin.TheApp.Screen(i)
		if scp, ok := p.ScreenPrefs[sc.Name]; ok {
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(ZoomFactor*scp.LogicalDPIScale, sc.PhysicalDPI)
		} else {
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(ZoomFactor*p.LogicalDPIScale, sc.PhysicalDPI)
		}
	}
	for _, w := range AllWindows {
		w.OSWin.SetLogicalDPI(w.OSWin.Screen().LogicalDPI)
	}
}

// Update everything with current preferences -- triggers rebuild of default styles
func (p *Preferences) Update() {
	ZoomFactor = 1 // reset so saved dpi is used
	p.Apply()

	RebuildDefaultStyles = true
	for _, w := range AllWindows {
		w.FullReRender()
	}
	RebuildDefaultStyles = false
	// needs another pass through to get it right..
	for _, w := range AllWindows {
		w.FullReRender()
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

// ScreenInfo returns screen info for all screens on the console
func (p *Preferences) ScreenInfo() string {
	ns := oswin.TheApp.NScreens()
	scinfo := ""
	for i := 0; i < ns; i++ {
		sc := oswin.TheApp.Screen(i)
		if i > 0 {
			scinfo += "<br><br>\n"
		}
		scinfo += fmt.Sprintf("Screen number: %v name: %v\n<br>    geom: %v, depth: %v, logical DPI: %v, physical DPI: %v, logical DPI scale: %v, physical size: %v\n<br>    device pixel ratio: %v, refresh rate: %v\n<br>    orientation: %v, native orientation: %v, primary orientation: %v\n", i, sc.Name, sc.Geometry, sc.Depth, sc.LogicalDPI, sc.PhysicalDPI, sc.LogicalDPI/sc.PhysicalDPI, sc.PhysicalSize, sc.DevicePixelRatio, sc.RefreshRate, sc.Orientation, sc.NativeOrientation, sc.PrimaryOrientation)
	}
	return scinfo
}

// SaveScreenZoom saves the current LogicalDPI scaling to name of current screen.
func (p *Preferences) SaveScreenZoom() {
	sc := oswin.TheApp.Screen(0)
	sp, ok := p.ScreenPrefs[sc.Name]
	if !ok {
		sp = ScreenPrefs{}
	}
	sp.LogicalDPIScale = sc.LogicalDPI / sc.PhysicalDPI
	if p.ScreenPrefs == nil {
		p.ScreenPrefs = make(map[string]ScreenPrefs)
	}
	p.ScreenPrefs[sc.Name] = sp
}

// DeleteSavedWindowGeoms deletes the file that saves the position and size of
// each window, by screen, and clear current in-memory cache.  You shouldn't
// need to use this but sometimes useful for testing.
func (p *Preferences) DeleteSavedWindowGeoms() {
	WinGeomPrefs.DeleteAll()
}

// Load colors from a JSON-formatted file.
func (p *ColorPrefs) LoadJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, p)
}

// Save colors to a JSON-formatted file.
func (p *ColorPrefs) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
	}
	return err
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

////////////////////////////////////////////////////////////////////////////////
//  FilePaths

type FilePaths []string

var SavedPaths FilePaths

// Load file paths from a JSON-formatted file.
func (p *FilePaths) LoadJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, p)
}

// Save file paths to a JSON-formatted file.
func (p *FilePaths) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	if err != nil {
		// PromptDialog(nil, "Could not Save to File", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
	}
	return err
}

// AddPath inserts a path to the file paths (at the start), subject to max
// length -- if path is already on the list then it is moved to the start.
func (p *FilePaths) AddPath(path string, max int) {
	sz := len(*p)

	if sz > max {
		*p = (*p)[:max]
	}

	for i, s := range *p {
		if s == path {
			if i == 0 {
				return
			}
			copy((*p)[1:i+1], (*p)[0:i])
			(*p)[0] = path
			return
		}
	}

	if sz >= max {
		copy((*p)[1:max], (*p)[0:max-1])
		(*p)[0] = path
	} else {
		*p = append(*p, "")
		if sz > 0 {
			copy((*p)[1:], (*p)[0:sz])
		}
		(*p)[0] = path
	}
}

// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
var SavedPathsFileName = "saved_paths.json"

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
}

// LoadPaths loads the active SavedPaths from prefs dir
func LoadPaths() {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.LoadJSON(pnm)
}
