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
	"github.com/goki/ki/kit"
)

// FileName is used to specify an filename (including path) -- automtically
// opens the FileView dialog using ValueView system.  Use this for any method
// args that are filenames to trigger use of FileViewDialog under MethView
// automatic method calling.
type FileName string

// ZoomFactor is a temporary multiplier on LogicalDPI used for per-session
// display zooming without changing prefs -- see SaveZoom to save prefs
// with this current factor.
var ZoomFactor = float32(1.0)

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences.
type ScreenPrefs struct {
	LogicalDPIScale float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc.  Actual Logical DPI is enforced to be a multiple of 6, so the precise number here isn't critical -- rounding to 2 digits is more than sufficient."`
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
	KeyMap          KeyMapName             `desc:"select the active keymap from list of available keymaps -- see Edit KeyMaps for editing / saving / loading that list"`
	SaveKeyMaps     bool                   `desc:"if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords"`
	PrefsOverride   bool                   `desc:"if true my custom style preferences override other styling -- otherwise they provide defaults that can be overriden by app-specific styling"`
	CustomStyles    ki.Props               `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	FontFamily      FontName               `desc:"default font family when otherwise not specified"`
	FontPaths       []string               `desc:"extra font paths, beyond system defaults -- searched first"`
	FavPaths        FavPaths               `desc:"favorite paths, shown in FileViewer and also editable there"`
	SavedPathsMax   int                    `desc:"maximum number of saved paths to save in FileView"`
	FileViewSort    string                 `view:"-" desc:"column to sort by in FileView, and :up or :down for direction -- updated automatically via FileView"`
	ColorFilename   FileName               `view:"-" ext:".json" desc:"filename for saving / loading colors"`
	Changed         bool                   `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

// Prefs are the overall preferences
var Prefs = Preferences{}

func (pf *ColorPrefs) Defaults() {
	pf.Font.SetColor(color.Black)
	pf.Border.SetString("#666", nil)
	pf.Background.SetColor(color.White)
	pf.Shadow.SetString("darker-10", &pf.Background)
	pf.Control.SetString("#EEF", nil)
	pf.Icon.SetString("highlight-30", pf.Control)
	pf.Select.SetString("#CFC", nil)
	pf.Highlight.SetString("#FFA", nil)
	pf.Link.SetString("#00F", nil)
}

// PrefColor returns preference color of given name (case insensitive)
func (pf *ColorPrefs) PrefColor(clrName string) *Color {
	lc := strings.Replace(strings.ToLower(clrName), "-", "", -1)
	switch lc {
	case "font":
		return &pf.Font
	case "background":
		return &pf.Background
	case "shadow":
		return &pf.Shadow
	case "border":
		return &pf.Border
	case "control":
		return &pf.Control
	case "icon":
		return &pf.Icon
	case "select":
		return &pf.Select
	case "highlight":
		return &pf.Highlight
	case "link":
		return &pf.Link
	}
	log.Printf("Preference color %v (simlified to: %v) not found\n", clrName, lc)
	return nil
}

func (pf *ParamPrefs) Defaults() {
	pf.DoubleClickMSec = 500
	pf.ScrollWheelRate = 20
	pf.LocalMainMenu = false
}

func (pf *Preferences) Defaults() {
	pf.LogicalDPIScale = 1.0
	pf.Colors.Defaults()
	pf.Params.Defaults()
	pf.FavPaths.SetToDefaults()
	pf.FontFamily = "Go"
	pf.SavedPathsMax = 20
	pf.KeyMap = DefaultKeyMap
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "prefs.json"

// Open preferences from GoGi standard prefs directory
func (pf *Preferences) Open() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err) // ok to be non-existant
		return err
	}
	err = json.Unmarshal(b, pf)
	if pf.SaveKeyMaps {
		AvailKeyMaps.OpenPrefs()
	}
	pf.Changed = false
	return err
}

// Save Preferences to GoGi standard prefs directory
func (pf *Preferences) Save() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	if pf.SaveKeyMaps {
		AvailKeyMaps.SavePrefs()
	}
	pf.Changed = false
	return err
}

// OpenColors colors from a JSON-formatted file.
func (pf *Preferences) OpenColors(filename FileName) error {
	err := pf.Colors.OpenJSON(filename)
	if err == nil {
		pf.Update()
	}
	pf.Changed = true
	return err
}

// Save colors to a JSON-formatted file, for easy sharing of your favorite
// palettes.
func (pf *Preferences) SaveColors(filename FileName) error {
	pf.Changed = true
	return pf.Colors.SaveJSON(filename)
}

// Apply preferences to all the relevant settings.
func (pf *Preferences) Apply() {
	np := len(pf.FavPaths)
	for i := 0; i < np; i++ {
		if pf.FavPaths[i].Ic == "" {
			pf.FavPaths[i].Ic = "folder"
		}
	}

	mouse.DoubleClickMSec = pf.Params.DoubleClickMSec
	mouse.ScrollWheelRate = pf.Params.ScrollWheelRate
	LocalMainMenu = pf.Params.LocalMainMenu

	if pf.KeyMap != "" {
		SetActiveKeyMapName(pf.KeyMap) // fills in missing pieces
	}
	if pf.FontPaths != nil {
		paths := append(pf.FontPaths, oswin.TheApp.FontPaths()...)
		FontLibrary.InitFontPaths(paths...)
	} else {
		FontLibrary.InitFontPaths(oswin.TheApp.FontPaths()...)
	}
	pf.ApplyDPI()
}

// ApplyDPI updates the screen LogicalDPI values according to current
// preferences and zoom factor, and then updates all open windows as well.
func (pf *Preferences) ApplyDPI() {
	n := oswin.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := oswin.TheApp.Screen(i)
		if scp, ok := pf.ScreenPrefs[sc.Name]; ok {
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(ZoomFactor*scp.LogicalDPIScale, sc.PhysicalDPI)
		} else {
			sc.LogicalDPI = oswin.LogicalFmPhysicalDPI(ZoomFactor*pf.LogicalDPIScale, sc.PhysicalDPI)
		}
	}
	for _, w := range AllWindows {
		w.OSWin.SetLogicalDPI(w.OSWin.Screen().LogicalDPI)
	}
}

// Update updates all open windows with current preferences -- triggers
// rebuild of default styles.
func (pf *Preferences) Update() {
	ZoomFactor = 1 // reset so saved dpi is used
	pf.Apply()

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

// ScreenInfo returns screen info for all screens on the console.
func (pf *Preferences) ScreenInfo() string {
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

// SaveZoom saves the current LogicalDPI scaling, either as the overall
// default or specific to the current screen.
func (pf *Preferences) SaveZoom(forCurrentScreen bool) {
	sc := oswin.TheApp.Screen(0)
	if forCurrentScreen {
		sp, ok := pf.ScreenPrefs[sc.Name]
		if !ok {
			sp = ScreenPrefs{}
		}
		sp.LogicalDPIScale = Truncate32(sc.LogicalDPI/sc.PhysicalDPI, 2)
		if pf.ScreenPrefs == nil {
			pf.ScreenPrefs = make(map[string]ScreenPrefs)
		}
		pf.ScreenPrefs[sc.Name] = sp
	} else {
		pf.LogicalDPIScale = Truncate32(sc.LogicalDPI/sc.PhysicalDPI, 2)
	}
	pf.Changed = true
}

// DeleteSavedWindowGeoms deletes the file that saves the position and size of
// each window, by screen, and clear current in-memory cache.  You shouldn't
// need to use this but sometimes useful for testing.
func (pf *Preferences) DeleteSavedWindowGeoms() {
	WinGeomPrefs.DeleteAll()
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
func (pf *Preferences) EditKeyMaps() {
	pf.SaveKeyMaps = true
	pf.Changed = true
	TheViewIFace.KeyMapsView(&AvailKeyMaps)
}

// OpenJSON opens colors from a JSON-formatted file.
func (pf *ColorPrefs) OpenJSON(filename FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		PromptDialog(nil, DlgOpts{Title: "File Not Found", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, pf)
}

// SaveJSON saves colors to a JSON-formatted file.
func (pf *ColorPrefs) SaveJSON(filename FileName) error {
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err) // unlikely
		return err
	}
	err = ioutil.WriteFile(string(filename), b, 0644)
	if err != nil {
		PromptDialog(nil, DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, true, false, nil, nil)
		log.Println(err)
	}
	return err
}

// PreferencesProps define the ToolBar and MenuBar for StructView, e.g., giv.PrefsView
var PreferencesProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Update", ki.Props{
				"shortcut": "Command+U",
			}},
			{"Open", ki.Props{
				"shortcut": "Command+O",
			}},
			{"Save", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(pfi interface{}, act *Action) {
					pf := pfi.(*Preferences)
					act.SetActiveState(pf.Changed)
				},
			}},
			{"sep-color", ki.BlankProp{}},
			{"OpenColors", ki.Props{
				"Args": ki.PropSlice{
					{"Color File Name", ki.Props{
						"default-field": "ColorFilename",
						"ext":           ".json",
					}},
				},
			}},
			{"SaveColors", ki.Props{
				"Args": ki.PropSlice{
					{"Color File Name", ki.Props{
						"default-field": "ColorFilename",
						"ext":           ".json",
					}},
				},
			}},
			{"sep-misc", ki.BlankProp{}},
			{"SaveZoom", ki.Props{
				"desc": "Save current zoom magnification factor, either for all screens or for the current screen only",
				"Args": ki.PropSlice{
					{"For Current Screen Only?", ki.Props{
						"desc": "click this to save zoom specifically for current screen",
					}},
				},
			}},
			{"DeleteSavedWindowGeoms", ki.Props{
				"confirm": true,
				"desc":    "Are you <i>sure</i>?  This deletes the file that saves the position and size of each window, by screen, and clear current in-memory cache.  You shouldn't generally need to do this but sometimes it is useful for testing or windows are showing up in bad places that you can't recover from.",
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"Update", ki.Props{
			"desc": "Updates all open windows with current preferences -- triggers rebuild of default styles.",
			"icon": "update",
		}},
		{"sep-file", ki.BlankProp{}},
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs.json file, which is auto-loaded at startup.",
			"icon": "file-save",
			"updtfunc": func(pfi interface{}, act *Action) {
				pf := pfi.(*Preferences)
				act.SetActiveStateUpdt(pf.Changed)
			},
		}},
		{"sep-color", ki.BlankProp{}},
		{"Colors", ki.PropSlice{ // sub-menu
			{"OpenColors", ki.Props{
				"icon": "file-open",
				"Args": ki.PropSlice{
					{"Color File Name", ki.Props{
						"default-field": "ColorFilename",
						"ext":           ".json",
					}},
				},
			}},
			{"SaveColors", ki.Props{
				"icon": "file-save",
				"Args": ki.PropSlice{
					{"Color File Name", ki.Props{
						"default-field": "ColorFilename",
						"ext":           ".json",
					}},
				},
			}},
		}},
		{"sep-scrn", ki.BlankProp{}},
		{"SaveZoom", ki.Props{
			"icon": "zoom-in",
			"desc": "Save current zoom magnification factor, either for all screens or for the current screen only",
			"Args": ki.PropSlice{
				{"For Current Screen Only?", ki.Props{
					"desc": "click this to save zoom specifically for current screen",
				}},
			},
		}},
		{"ScreenInfo", ki.Props{
			"desc":        "shows parameters about all the active screens",
			"icon":        "info",
			"show-return": true,
		}},
		{"sep-key", ki.BlankProp{}},
		{"EditKeyMaps", ki.Props{
			"icon": "keyboard",
			"desc": "opens the KeyMapsView editor to create new keymaps / save / load from other files, etc.  Current keymaps are saved and loaded with preferences automatically if SaveKeyMaps is clicked (will be turned on automatically if you open this editor).",
		},
		},
	},
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
func (pf *FavPaths) SetToDefaults() {
	*pf = make(FavPaths, len(DefaultPaths))
	copy(*pf, DefaultPaths)
}

// FindPath returns index of path on list, or -1, false if not found
func (pf *FavPaths) FindPath(path string) (int, bool) {
	for i, fi := range *pf {
		if fi.Path == path {
			return i, true
		}
	}
	return -1, false
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

// Open file paths from a JSON-formatted file.
func (pf *FilePaths) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), true, false, nil, nil, nil)
		log.Println(err)
		return err
	}
	return json.Unmarshal(b, pf)
}

// Save file paths to a JSON-formatted file.
func (pf *FilePaths) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(pf, "", "  ")
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
func (pf *FilePaths) AddPath(path string, max int) {
	sz := len(*pf)

	if sz > max {
		*pf = (*pf)[:max]
	}

	for i, s := range *pf {
		if s == path {
			if i == 0 {
				return
			}
			copy((*pf)[1:i+1], (*pf)[0:i])
			(*pf)[0] = path
			return
		}
	}

	if sz >= max {
		copy((*pf)[1:max], (*pf)[0:max-1])
		(*pf)[0] = path
	} else {
		*pf = append(*pf, "")
		if sz > 0 {
			copy((*pf)[1:], (*pf)[0:sz])
		}
		(*pf)[0] = path
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

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
}
