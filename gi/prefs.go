// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
	"github.com/goki/pi/langs/golang"
)

// ZoomFactor is a temporary multiplier on LogicalDPI used for per-session
// display zooming without changing prefs -- see SaveZoom to save prefs
// with this current factor.
var ZoomFactor = float32(1.0)

// Preferences are the overall user preferences for GoGi, providing some basic
// customization -- in addition, most gui settings can be styled using
// CSS-style sheets under CustomStyle.  These prefs are saved and loaded from
// the GoGi user preferences directory -- see oswin/App for further info.
type Preferences struct {
	LogicalDPIScale      float32                `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc"`
	ScreenPrefs          map[string]ScreenPrefs `desc:"screen-specific preferences -- will override overall defaults if set"`
	Colors               ColorPrefs             `desc:"active color preferences"`
	ColorSchemes         map[string]*ColorPrefs `desc:"named color schemes -- has Light and Dark schemes by default"`
	Params               ParamPrefs             `view:"inline" desc:"parameters controlling GUI behavior"`
	Editor               EditorPrefs            `view:"inline" desc:"editor preferences -- for TextView etc"`
	KeyMap               KeyMapName             `desc:"select the active keymap from list of available keymaps -- see Edit KeyMaps for editing / saving / loading that list"`
	SaveKeyMaps          bool                   `desc:"if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords"`
	SaveDetailed         bool                   `desc:"if set, the detailed preferences are saved and loaded at startup -- only "`
	CustomStyles         ki.Props               `desc:"a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive"`
	CustomStylesOverride bool                   `desc:"if true my custom styles override other styling (i.e., they come <i>last</i> in styling process -- otherwise they provide defaults that can be overridden by app-specific styling (i.e, they come first)."`
	FontFamily           FontName               `desc:"default font family when otherwise not specified"`
	MonoFont             FontName               `desc:"default mono-spaced font family"`
	FontPaths            []string               `desc:"extra font paths, beyond system defaults -- searched first"`
	User                 User                   `desc:"user info -- partially filled-out automatically if empty / when prefs first created"`
	FavPaths             FavPaths               `desc:"favorite paths, shown in FileViewer and also editable there"`
	FileViewSort         string                 `view:"-" desc:"column to sort by in FileView, and :up or :down for direction -- updated automatically via FileView"`
	ColorFilename        FileName               `view:"-" ext:".json" desc:"filename for saving / loading colors"`
	Changed              bool                   `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_Preferences = kit.Types.AddType(&Preferences{}, PreferencesProps)

// Prefs are the overall preferences
var Prefs = Preferences{}

func init() {
	gist.ThePrefs = &Prefs
}

func (pf *Preferences) Defaults() {
	pf.LogicalDPIScale = 1.0
	pf.Colors.Defaults()
	pf.ColorSchemes = DefaultColorSchemes()
	pf.Params.Defaults()
	pf.Editor.Defaults()
	pf.FavPaths.SetToDefaults()
	pf.FontFamily = "Go"
	pf.MonoFont = "Go Mono"
	pf.KeyMap = DefaultKeyMap
	pf.UpdateUser()
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "prefs.json"

// Open preferences from GoGi standard prefs directory
func (pf *Preferences) Open() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err) // ok to be non-existent
		return err
	}
	err = json.Unmarshal(b, pf)
	if pf.SaveKeyMaps {
		err = AvailKeyMaps.OpenPrefs()
		if err != nil {
			pf.SaveKeyMaps = false
		}
	}
	if pf.SaveDetailed {
		PrefsDet.Open()
	}
	if pf.User.Username == "" {
		pf.UpdateUser()
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
	if pf.SaveDetailed {
		PrefsDet.Save()
	}
	pf.Changed = false
	return err
}

// IsDarkMode returns true if the current background color preference is dark
func (pf *Preferences) IsDarkMode() bool {
	return pf.Colors.Background.IsDark()
}

// OpenColors colors from a JSON-formatted file.
func (pf *Preferences) OpenColors(filename FileName) error {
	err := pf.Colors.OpenJSON(filename)
	// if err == nil {
	// 	pf.UpdateAll() // no!  this recolors the dialog as it is closing!  do it separately
	// }
	pf.Changed = true
	return err
}

// Save colors to a JSON-formatted file, for easy sharing of your favorite
// palettes.
func (pf *Preferences) SaveColors(filename FileName) error {
	pf.Changed = true
	return pf.Colors.SaveJSON(filename)
}

// LightMode sets colors to light mode
func (pf *Preferences) LightMode() {
	lc, ok := pf.ColorSchemes["Light"]
	if !ok {
		log.Printf("Light ColorScheme not found\n")
		return
	}
	pf.Colors = *lc
	pf.Save()
	pf.UpdateAll()
}

// DarkMode sets colors to dark mode
func (pf *Preferences) DarkMode() {
	lc, ok := pf.ColorSchemes["Dark"]
	if !ok {
		log.Printf("Dark ColorScheme not found\n")
		return
	}
	pf.Colors = *lc
	pf.Save()
	pf.UpdateAll()
}

// Apply preferences to all the relevant settings.
func (pf *Preferences) Apply() {
	np := len(pf.FavPaths)
	for i := 0; i < np; i++ {
		if pf.FavPaths[i].Ic == "" {
			pf.FavPaths[i].Ic = "folder"
		}
	}
	if pf.Colors.HiStyle == "" {
		pf.Colors.HiStyle = "emacs"
	}
	if len(pf.ColorSchemes) < 2 {
		pf.ColorSchemes = DefaultColorSchemes()
	}
	if pf.ColorSchemes["Light"].HiStyle == "" {
		pf.ColorSchemes["Light"].HiStyle = "emacs"
	}
	if pf.ColorSchemes["Dark"].HiStyle == "" {
		pf.ColorSchemes["Dark"].HiStyle = "monokai"
	}

	TheViewIFace.SetHiStyleDefault(pf.Colors.HiStyle)
	mouse.DoubleClickMSec = pf.Params.DoubleClickMSec
	mouse.ScrollWheelSpeed = pf.Params.ScrollWheelSpeed
	LocalMainMenu = pf.Params.LocalMainMenu

	if pf.KeyMap != "" {
		SetActiveKeyMapName(pf.KeyMap) // fills in missing pieces
	}
	if pf.SaveDetailed {
		PrefsDet.Apply()
	}
	if pf.FontPaths != nil {
		paths := append(pf.FontPaths, oswin.TheApp.FontPaths()...)
		girl.FontLibrary.InitFontPaths(paths...)
	} else {
		girl.FontLibrary.InitFontPaths(oswin.TheApp.FontPaths()...)
	}
	pf.ApplyDPI()
}

// ApplyDPI updates the screen LogicalDPI values according to current
// preferences and zoom factor, and then updates all open windows as well.
func (pf *Preferences) ApplyDPI() {
	n := oswin.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := oswin.TheApp.Screen(i)
		if sc == nil {
			continue
		}
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

// UpdateAll updates all open windows with current preferences -- triggers
// rebuild of default styles.
func (pf *Preferences) UpdateAll() {
	ZoomFactor = 1 // reset so saved dpi is used
	pf.Apply()

	gist.RebuildDefaultStyles = true
	gist.ColorSpecCache = nil
	gist.StyleTemplates = nil
	// for _, w := range AllWindows {  // no need and just messes stuff up!
	// 	w.SetSize(w.OSWin.Size())
	// }
	// needs another pass through to get it right..
	for _, w := range AllWindows {
		w.FullReRender()
	}
	gist.RebuildDefaultStyles = false
	// and another without rebuilding?  yep all are required
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
		scinfo += fmt.Sprintf("Screen number: %v Name: %v<br>\n    Geom: %v, DevPixRatio: %v<br>\n    Pixels: %v, Physical size: %v mm<br>\n    Logical DPI: %v, Physical DPI: %v, Logical DPI scale: %v<br>\n    Depth: %v, Refresh rate: %v<br>\n    Orientation: %v, Native orientation: %v, Primary orientation: %v<br>\n", i, sc.Name, sc.Geometry, sc.DevicePixelRatio, sc.PixSize, sc.PhysicalSize, sc.LogicalDPI, sc.PhysicalDPI, sc.LogicalDPI/sc.PhysicalDPI, sc.Depth, sc.RefreshRate, sc.Orientation, sc.NativeOrientation, sc.PrimaryOrientation)
	}
	return scinfo
}

// VersionInfo returns GoGi version information
func (pf *Preferences) VersionInfo() string {
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
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
		sp.LogicalDPIScale = mat32.Truncate(sc.LogicalDPI/sc.PhysicalDPI, 2)
		if pf.ScreenPrefs == nil {
			pf.ScreenPrefs = make(map[string]ScreenPrefs)
		}
		pf.ScreenPrefs[sc.Name] = sp
	} else {
		pf.LogicalDPIScale = mat32.Truncate(sc.LogicalDPI/sc.PhysicalDPI, 2)
	}
	pf.Save()
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

// EditHiStyles opens the HiStyleView editor to customize highlighting styles
func (pf *Preferences) EditHiStyles() {
	TheViewIFace.HiStylesView(false) // false = custom
}

// EditDetailed opens the PrefsDetView editor to edit detailed params
func (pf *Preferences) EditDetailed() {
	pf.SaveDetailed = true
	pf.Changed = true
	TheViewIFace.PrefsDetView(&PrefsDet)
}

// EditDebug opens the PrefsDbgView editor to edit debugging params
func (pf *Preferences) EditDebug() {
	TheViewIFace.PrefsDbgView(&PrefsDbg)
}

// UpdateUser gets the user info from the OS
func (pf *Preferences) UpdateUser() {
	usr, err := user.Current()
	if err == nil {
		pf.User.User = *usr
	}
}

/////////////////////////////////////////////////////////
// Following are gist.Prefs interface

// PrefColor returns preference color of given name (case insensitive)
// std names are: font, background, shadow, border, control, icon, select, highlight, link
func (pf *Preferences) PrefColor(clrName string) *gist.Color {
	return pf.Colors.PrefColor(clrName)
}

// PrefFontFamily returns the default FontFamily
func (pf *Preferences) PrefFontFamily() string {
	return string(pf.FontFamily)
}

// PreferencesProps define the ToolBar and MenuBar for StructView, e.g., giv.PrefsView
var PreferencesProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"UpdateAll", ki.Props{}},
			{"Open", ki.Props{
				"shortcut": KeyFunMenuOpen,
			}},
			{"Save", ki.Props{
				"shortcut": KeyFunMenuSave,
				"updtfunc": func(pfi interface{}, act *Action) {
					pf := pfi.(*Preferences)
					act.SetActiveState(pf.Changed)
				},
			}},
			{"sep-color", ki.BlankProp{}},
			{"LightMode", ki.Props{}},
			{"DarkMode", ki.Props{}},
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
		{"UpdateAll", ki.Props{
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
		{"LightMode", ki.Props{
			"desc": "Set color mode to Light mode as defined in ColorSchemes -- automatically does Save and UpdateAll ",
			"icon": "color",
		}},
		{"DarkMode", ki.Props{
			"desc": "Set color mode to Dark mode as defined in ColorSchemes -- automatically does Save and UpdateAll",
			"icon": "color",
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
		{"VersionInfo", ki.Props{
			"desc":        "shows current GoGi version information",
			"icon":        "info",
			"show-return": true,
		}},
		{"sep-key", ki.BlankProp{}},
		{"EditKeyMaps", ki.Props{
			"icon": "keyboard",
			"desc": "opens the KeyMapsView editor to create new keymaps / save / load from other files, etc.  Current keymaps are saved and loaded with preferences automatically if SaveKeyMaps is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditHiStyles", ki.Props{
			"icon": "file-binary",
			"desc": "opens the HiStylesView editor of highlighting styles.",
		}},
		{"EditDetailed", ki.Props{
			"icon": "file-binary",
			"desc": "opens the PrefsDetView editor to edit detailed params that are not typically user-modified, but can be if you really care..  Turns on the SaveDetailed flag so these will be saved and loaded automatically -- can toggle that back off if you don't actually want to.",
		}},
		{"EditDebug", ki.Props{
			"icon": "file-binary",
			"desc": "Opens the PrefsDbgView editor to control debugging parameters. These are not saved -- only set dynamically during running.",
		}},
	},
}

/////////////////////////////////////////////////////////////////////////////////
//   ColorPrefs

// ColorPrefs specify colors for all major categories of GUI elements, and are
// used in the default styles.
type ColorPrefs struct {
	HiStyle    HiStyleName `desc:"text highilighting style / theme"`
	Font       gist.Color  `desc:"default font / pen color"`
	Background gist.Color  `desc:"default background color"`
	Shadow     gist.Color  `desc:"color for shadows -- should generally be a darker shade of the background color"`
	Border     gist.Color  `desc:"default border color, for button, frame borders, etc"`
	Control    gist.Color  `desc:"default main color for controls: buttons, etc"`
	Icon       gist.Color  `desc:"color for icons or other solidly-colored, small elements"`
	Select     gist.Color  `desc:"color for selected elements"`
	Highlight  gist.Color  `desc:"color for highlight background"`
	Link       gist.Color  `desc:"color for links in text etc"`
}

var KiT_ColorPrefs = kit.Types.AddType(&ColorPrefs{}, ColorPrefsProps)

func (pf *ColorPrefs) Defaults() {
	pf.HiStyle = "emacs"
	pf.Font.SetColor(gist.Black)
	pf.Border.SetString("#666", nil)
	pf.Background.SetColor(gist.White)
	pf.Shadow.SetString("darker-10", &pf.Background)
	pf.Control.SetString("#F8F8F8", nil)
	pf.Icon.SetString("highlight-30", pf.Control)
	pf.Select.SetString("#CFC", nil)
	pf.Highlight.SetString("#FFA", nil)
	pf.Link.SetString("#00F", nil)
}

func (pf *ColorPrefs) DarkDefaults() {
	pf.HiStyle = "monokai"
	pf.Font.SetUInt8(175, 175, 175, 255)
	pf.Background.SetUInt8(0, 0, 0, 255)
	pf.Shadow.SetUInt8(64, 64, 64, 255)
	pf.Border.SetUInt8(102, 102, 102, 255)
	pf.Control.SetUInt8(17, 57, 57, 255)
	pf.Icon.SetUInt8(70, 70, 192, 255)
	pf.Select.SetUInt8(17, 100, 100, 255)
	pf.Highlight.SetUInt8(66, 82, 0, 255)
	pf.Link.SetUInt8(117, 117, 249, 255)
}

func DefaultColorSchemes() map[string]*ColorPrefs {
	cs := map[string]*ColorPrefs{}
	lc := &ColorPrefs{}
	lc.Defaults()
	cs["Light"] = lc
	dc := &ColorPrefs{}
	dc.DarkDefaults()
	cs["Dark"] = dc
	return cs
}

// PrefColor returns preference color of given name (case insensitive)
// std names are: font, background, shadow, border, control, icon, select, highlight, link
func (pf *ColorPrefs) PrefColor(clrName string) *gist.Color {
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
	log.Printf("Preference color %v (simplified to: %v) not found\n", clrName, lc)
	return nil
}

// OpenJSON opens colors from a JSON-formatted file.
func (pf *ColorPrefs) OpenJSON(filename FileName) error {
	b, err := ioutil.ReadFile(string(filename))
	if err != nil {
		PromptDialog(nil, DlgOpts{Title: "File Not Found", Prompt: err.Error()}, AddOk, NoCancel, nil, nil)
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
		PromptDialog(nil, DlgOpts{Title: "Could not Save to File", Prompt: err.Error()}, AddOk, NoCancel, nil, nil)
		log.Println(err)
	}
	return err
}

// SetToPrefs sets this color scheme as the current active setting in overall
// default prefs.
func (pf *ColorPrefs) SetToPrefs() {
	Prefs.Colors = *pf
	Prefs.UpdateAll()
}

// ColorPrefsProps defines the ToolBar
var ColorPrefsProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"OpenJSON", ki.Props{
			"label": "Open...",
			"icon":  "file-open",
			"desc":  "open set of colors from a json-formatted file",
			"Args": ki.PropSlice{
				{"Color File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SaveJSON", ki.Props{
			"label": "Save As...",
			"desc":  "Saves colors to JSON formatted file.",
			"icon":  "file-save",
			"Args": ki.PropSlice{
				{"Color File Name", ki.Props{
					"ext": ".json",
				}},
			},
		}},
		{"SetToPrefs", ki.Props{
			"desc": "Sets this color scheme as the current active color scheme in Prefs.",
			"icon": "reset",
		}},
	},
}

//////////////////////////////////////////////////////////////////
//  ParamPrefs

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences.
type ScreenPrefs struct {
	LogicalDPIScale float32 `min:"0.1" step:"0.1" desc:"overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc.  Actual Logical DPI is enforced to be a multiple of 6, so the precise number here isn't critical -- rounding to 2 digits is more than sufficient."`
}

// ParamPrefs contains misc parameters controlling GUI behavior.
type ParamPrefs struct {
	DoubleClickMSec  int     `min:"100" step:"50" desc:"the maximum time interval in msec between button press events to count as a double-click"`
	ScrollWheelSpeed float32 `min:"0.01" step:"1" desc:"how fast the scroll wheel moves -- typically pixels per wheel step but units can be arbitrary.  It is generally impossible to standardize speed and variable across devices, and we don't have access to the system settings, so unfortunately you have to set it here."`
	LocalMainMenu    bool    `desc:"controls whether the main menu is displayed locally at top of each window, in addition to global menu at the top of the screen.  Mac native apps do not do this, but OTOH it makes things more consistent with other platforms, and with larger screens, it can be convenient to have access to all the menu items right there."`
	BigFileSize      int     `def:"10000000" desc:"the limit of file size, above which user will be prompted before opening / copying, etc."`
	SavedPathsMax    int     `desc:"maximum number of saved paths to save in FileView"`
	Smooth3D         bool    `desc:"turn on smoothing in 3D rendering -- this should be on by default but if you get an error telling you to turn it off, then do so (because your hardware can't handle it)"`
}

func (pf *ParamPrefs) Defaults() {
	pf.DoubleClickMSec = 500
	pf.ScrollWheelSpeed = 20
	pf.LocalMainMenu = true // much better
	pf.BigFileSize = 10000000
	pf.SavedPathsMax = 50
	pf.Smooth3D = true
}

// User basic user information that might be needed for different apps
type User struct {
	user.User
	Email string `desc:"default email address -- e.g., for recording changes in a version control system"`
}

//////////////////////////////////////////////////////////////////
//  EditorPrefs

// EditorPrefs contains editor preferences.  It can also be set
// from ki.Props style properties.
type EditorPrefs struct {
	TabSize      int  `xml:"tab-size" desc:"size of a tab, in chars -- also determines indent level for space indent"`
	SpaceIndent  bool `xml:"space-indent" desc:"use spaces for indentation, otherwise tabs"`
	WordWrap     bool `xml:"word-wrap" desc:"wrap lines at word boundaries -- otherwise long lines scroll off the end"`
	LineNos      bool `xml:"line-nos" desc:"show line numbers"`
	Completion   bool `xml:"completion" desc:"use the completion system to suggest options while typing"`
	SpellCorrect bool `xml:"spell-correct" desc:"suggest corrections for unknown words while typing"`
	AutoIndent   bool `xml:"auto-indent" desc:"automatically indent lines when enter, tab, }, etc pressed"`
	EmacsUndo    bool `xml:"emacs-undo" desc:"use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo"`
	DepthColor   bool `xml:"depth-color" desc:"colorize the background according to nesting depth"`
}

// Defaults are the defaults for EditorPrefs
func (pf *EditorPrefs) Defaults() {
	pf.TabSize = 4
	pf.WordWrap = true
	pf.LineNos = true
	pf.Completion = true
	pf.SpellCorrect = true
	pf.AutoIndent = true
	pf.DepthColor = true
}

// StyleFromProps styles Slider-specific fields from ki.Prop properties
// doesn't support inherit or default
func (pf *EditorPrefs) StyleFromProps(props ki.Props) {
	for key, val := range props {
		if len(key) == 0 {
			continue
		}
		if key[0] == '#' || key[0] == '.' || key[0] == ':' || key[0] == '_' {
			continue
		}
		switch key {
		case "tab-size":
			if iv, ok := kit.ToInt(val); ok {
				pf.TabSize = int(iv)
			}
		case "space-indent":
			if iv, ok := kit.ToBool(val); ok {
				pf.SpaceIndent = iv
			}
		case "word-wrap":
			if iv, ok := kit.ToBool(val); ok {
				pf.WordWrap = iv
			}
		case "line-nos":
			if iv, ok := kit.ToBool(val); ok {
				pf.LineNos = iv
			}
		case "completion":
			if iv, ok := kit.ToBool(val); ok {
				pf.Completion = iv
			}
		case "spell-correct":
			if iv, ok := kit.ToBool(val); ok {
				pf.SpellCorrect = iv
			}
		case "auto-indent":
			if iv, ok := kit.ToBool(val); ok {
				pf.AutoIndent = iv
			}
		case "emacs-undo":
			if iv, ok := kit.ToBool(val); ok {
				pf.EmacsUndo = iv
			}
		case "depth-color":
			if iv, ok := kit.ToBool(val); ok {
				pf.DepthColor = iv
			}
		}
	}
}

//////////////////////////////////////////////////////////////////
//  FavoritePaths

// FavPathItem represents one item in a favorite path list, for display of
// favorites.  Is an ordered list instead of a map because user can organize
// in order
type FavPathItem struct {
	Ic   IconName `desc:"icon for item"`
	Name string   `width:"20" desc:"name of the favorite item"`
	Path string   `tableview:"-select"`
}

// Label satisfies the Labeler interface
func (fi FavPathItem) Label() string {
	return fi.Name
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

//////////////////////////////////////////////////////////////////
//  FilePaths

type FilePaths []string

var SavedPaths FilePaths

// Open file paths from a JSON-formatted file.
func (pf *FilePaths) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		// PromptDialog(nil, "File Not Found", err.Error(), AddOk, NoCancel, nil, nil, nil)
		// log.Println(err)
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
		// PromptDialog(nil, "Could not Save to File", err.Error(), AddOk, NoCancel, nil, nil, nil)
		log.Println(err)
	}
	return err
}

// AddPath inserts a path to the file paths (at the start), subject to max
// length -- if path is already on the list then it is moved to the start.
func (pf *FilePaths) AddPath(path string, max int) {
	StringsInsertFirstUnique((*[]string)(pf), path, max)
}

// SavedPathsFileName is the name of the saved file paths file in GoGi prefs directory
var SavedPathsFileName = "saved_paths.json"

// FileViewResetPaths defines a string that is added as an item to the recents menu
var FileViewResetPaths = "<i>Reset Paths</i>"

// FileViewEditPaths defines a string that is added as an item to the recents menu
var FileViewEditPaths = "<i>Edit Paths...</i>"

// SavedPathsExtras are the reset and edit items we add to the recents menu
var SavedPathsExtras = []string{MenuTextSeparator, FileViewResetPaths, FileViewEditPaths}

// SavePaths saves the active SavedPaths to prefs dir
func SavePaths() {
	StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

//////////////////////////////////////////////////////////////////
//  PrefsDetailed

// PrefsDetailed are more detailed params not usually customized, but
// available for those who really care..
type PrefsDetailed struct {
	MenuMaxHeight              int  `def:"30" min:"5" step:"1" desc:"the maximum height of any menu popup panel in units of font height -- scroll bars are enforced beyond that size."`
	EventSkipLagMSec           int  `def:"50" min:"5" max:"1000" step:"5" desc:"the number of milliseconds of lag between the time the event was sent to the time it is being processed, above which a repeated event type (scroll, drag, resize) is skipped"`
	FilterLaggyKeyEvents       bool `def:"false" desc:"set to true to apply laggy filter to KeyEvents (normally excluded)"`
	DragStartMSec              int  `def:"50" min:"5" max:"1000" step:"5" desc:"the number of milliseconds to wait before initiating a regular mouse drag event (as opposed to a basic mouse.Press)"`
	DragStartPix               int  `def:"4" min:"0" max:"100" step:"1" desc:"the number of pixels that must be moved before initiating a regular mouse drag event (as opposed to a basic mouse.Press)"`
	DNDStartMSec               int  `def:"200" min:"5" max:"1000" step:"5" desc:"the number of milliseconds to wait before initiating a drag-n-drop event -- gotta drag it like you mean it"`
	DNDStartPix                int  `def:"20" min:"0" max:"100" step:"1" desc:"the number of pixels that must be moved before initiating a drag-n-drop event -- gotta drag it like you mean it"`
	HoverStartMSec             int  `def:"1000" min:"10" max:"10000" step:"10" desc:"the number of milliseconds to wait before initiating a hover event (e.g., for opening a tooltip)"`
	HoverMaxPix                int  `def:"5" min:"0" max:"1000" step:"1" desc:"the maximum number of pixels that mouse can move and still register a Hover event"`
	CompleteWaitMSec           int  `def:"500" min:"10" max:"10000" step:"10" desc:"the number of milliseconds to wait before offering completions"`
	CompleteMaxItems           int  `def:"25" min:"5" step:"1" desc:"the maximum number of completions offered in popup"`
	CursorBlinkMSec            int  `def:"500" min:"0" max:"1000" step:"5" desc:"number of milliseconds that cursor blinks on and off -- set to 0 to disable blinking"`
	LayoutAutoScrollDelayMSec  int  `def:"25" min:"1" step:"5" desc:"is amount of time to wait (in Milliseconds) before trying to autoscroll again"`
	LayoutPageSteps            int  `def:"10" min:"1" step:"1" desc:"number of steps to take in PageUp / Down events in terms of number of items"`
	LayoutFocusNameTimeoutMSec int  `def:"500" min:"0" max:"5000" step:"20" desc:"the number of milliseconds between keypresses to combine characters into name to search for within layout -- starts over after this delay"`
	LayoutFocusNameTabMSec     int  `def:"2000" min:"10" max:"10000" step:"100" desc:"the number of milliseconds since last focus name event to allow tab to focus on next element with same name."`
	DialogsSepWindow           bool `def:"true" desc:"open dialogs in separate windows -- else do as popups in main window"`
	TextViewClipHistMax        int  `def:"100" min:"0" max:"1000" step:"5" desc:"Maximum amount of clipboard history to retain"`
	TextBufMaxScopeLines       int  `def:"100" min:"10" step:"10" desc:"maximum number of lines to look for matching scope syntax (parens, brackets)"`
	TextBufDiffRevertLines     int  `def:"10000" min:"0" step:"1000" desc:"text buffer max lines to use diff-based revert to more quickly update e.g., after file has been reformatted"`
	TextBufDiffRevertDiffs     int  `def:"20" min:"0" step:"1" desc:"text buffer max diffs to use diff-based revert to more quickly update e.g., after file has been reformatted -- if too many differences, just revert"`
	TextBufMarkupDelayMSec     int  `def:"1000" min:"100" step:"100" desc:"number of milliseconds to wait before starting a new background markup process, after text changes within a single line (always does after line insertion / deletion)"`
	MapInlineLen               int  `def:"3" min:"2" step:"1" desc:"the number of map elements at or below which an inline representation of the map will be presented -- more convenient for small #'s of props"`
	StructInlineLen            int  `def:"6" min:"2" step:"1" desc:"the number of elemental struct fields at or below which an inline representation of the struct will be presented -- more convenient for small structs"`
	SliceInlineLen             int  `def:"6" min:"2" step:"1" desc:"the number of slice elements below which inline will be used"`
	Changed                    bool `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_PrefsDetailed = kit.Types.AddType(&PrefsDetailed{}, PrefsDetailedProps)

// PrefsDet are the overall detailed preferences
var PrefsDet = PrefsDetailed{}

// PrefsDetailedFileName is the name of the detailed preferences file in GoGi prefs directory
var PrefsDetailedFileName = "prefs_det.json"

// Open detailed preferences from GoGi standard prefs directory
func (pf *PrefsDetailed) Open() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsDetailedFileName)
	b, err := ioutil.ReadFile(pnm)
	if err != nil {
		// log.Println(err) // ok to be non-existent
		return err
	}
	err = json.Unmarshal(b, pf)
	pf.Changed = false
	return err
}

// Save detailed prefs to GoGi standard prefs directory
func (pf *PrefsDetailed) Save() error {
	pdir := oswin.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsDetailedFileName)
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		log.Println(err)
		return err
	}
	err = ioutil.WriteFile(pnm, b, 0644)
	if err != nil {
		log.Println(err)
	}
	pf.Changed = false
	return err
}

// Defaults gets current values of parameters, which are effectively
// defaults
func (pf *PrefsDetailed) Defaults() {
	pf.MenuMaxHeight = MenuMaxHeight
	pf.EventSkipLagMSec = EventSkipLagMSec
	pf.DragStartMSec = DragStartMSec
	pf.DragStartPix = DragStartPix
	pf.DNDStartMSec = DNDStartMSec
	pf.DNDStartPix = DNDStartPix
	pf.HoverStartMSec = HoverStartMSec
	pf.HoverMaxPix = HoverMaxPix
	pf.CompleteWaitMSec = CompleteWaitMSec
	pf.CompleteMaxItems = CompleteMaxItems
	pf.CursorBlinkMSec = CursorBlinkMSec
	pf.LayoutAutoScrollDelayMSec = LayoutAutoScrollDelayMSec
	pf.LayoutPageSteps = LayoutPageSteps
	pf.LayoutFocusNameTimeoutMSec = LayoutFocusNameTimeoutMSec
	pf.LayoutFocusNameTabMSec = LayoutFocusNameTabMSec
	pf.MenuMaxHeight = MenuMaxHeight
	pf.DialogsSepWindow = DialogsSepWindow
	TheViewIFace.PrefsDetDefaults(pf)
	// in giv:
	// TextViewClipHistMax
	// TextBuf*
	// MapInlineLen
	// StructInlineLen
	// SliceInlineLen
}

// Apply detailed preferences to all the relevant settings.
func (pf *PrefsDetailed) Apply() {
	MenuMaxHeight = pf.MenuMaxHeight
	EventSkipLagMSec = pf.EventSkipLagMSec
	DragStartMSec = pf.DragStartMSec
	DragStartPix = pf.DragStartPix
	DNDStartMSec = pf.DNDStartMSec
	DNDStartPix = pf.DNDStartPix
	HoverStartMSec = pf.HoverStartMSec
	HoverMaxPix = pf.HoverMaxPix
	CompleteWaitMSec = pf.CompleteWaitMSec
	CompleteMaxItems = pf.CompleteMaxItems
	CursorBlinkMSec = pf.CursorBlinkMSec
	LayoutFocusNameTimeoutMSec = pf.LayoutFocusNameTimeoutMSec
	LayoutFocusNameTabMSec = pf.LayoutFocusNameTabMSec
	MenuMaxHeight = pf.MenuMaxHeight
	DialogsSepWindow = pf.DialogsSepWindow
	TheViewIFace.PrefsDetApply(pf)
	// in giv:
	// TextViewClipHistMax = pf.TextViewClipHistMax
	// TextBuf*
	// MapInlineLen
	// StructInlineLen
	// SliceInlineLen
}

// PrefsDetailedProps define the ToolBar and MenuBar for StructView, e.g., giv.PrefsDetView
var PrefsDetailedProps = ki.Props{
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Apply", ki.Props{}},
			{"Open", ki.Props{
				"shortcut": KeyFunMenuOpen,
			}},
			{"Save", ki.Props{
				"shortcut": KeyFunMenuSave,
				"updtfunc": func(pfi interface{}, act *Action) {
					pf := pfi.(*PrefsDetailed)
					act.SetActiveState(pf.Changed)
				},
			}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"Window", "Windows"},
	},
	"ToolBar": ki.PropSlice{
		{"Apply", ki.Props{
			"desc": "Apply parameters to affect actual behavior.",
			"icon": "update",
		}},
		{"sep-file", ki.BlankProp{}},
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs_det.json file, which is auto-loaded at startup.",
			"icon": "file-save",
			"updtfunc": func(pfi interface{}, act *Action) {
				pf := pfi.(*PrefsDetailed)
				act.SetActiveStateUpdt(pf.Changed)
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////
//  PrefsDebug

// PrefsDebug are debugging params
type PrefsDebug struct {
	Update2DTrace *bool `desc:"reports trace of updates that trigger re-rendering (printfs to stdout)"`

	Render2DTrace *bool `desc:"reports trace of the nodes rendering (printfs to stdout)"`

	Layout2DTrace *bool `desc:"reports trace of all layouts (printfs to stdout)"`

	WinEventTrace *bool `desc:"reports trace of window events (printfs to stdout)"`

	WinPublishTrace *bool `desc:"reports the stack trace leading up to win publish events which are expensive -- wrap multiple updates in UpdateStart / End to prevent"`

	KeyEventTrace *bool `desc:"reports trace of keyboard events (printfs to stdout)"`

	EventTrace *bool `desc:"reports trace of event handling (printfs to stdout)"`

	DNDTrace *bool `desc:"reports trace of DND events handling"`

	GoCompleteTrace *bool `desc:"reports trace of Go language completion & lookup process"`

	GoTypeTrace *bool `desc:"reports trace of Go language type parsing and inference process"`

	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-" desc:"flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc."`
}

var KiT_PrefsDebug = kit.Types.AddType(&PrefsDebug{}, PrefsDebugProps)

// PrefsDbg are the overall debugging preferences
var PrefsDbg = PrefsDebug{}

// PrefsDebugProps define the ToolBar and MenuBar for StructView, e.g., giv.PrefsDbgView
var PrefsDebugProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"Profile", ki.Props{
			"desc": "Toggle profiling of program on or off -- does both targeted and global CPU and Memory profiling.",
			"icon": "update",
		}},
	},
}

// Connect connects debug fields with actual variables controlling debugging
func (pf *PrefsDebug) Connect() {
	pf.Update2DTrace = &Update2DTrace
	pf.Render2DTrace = &Render2DTrace
	pf.Layout2DTrace = &Layout2DTrace
	pf.WinEventTrace = &WinEventTrace
	pf.WinPublishTrace = &WinPublishTrace
	pf.KeyEventTrace = &KeyEventTrace
	pf.EventTrace = &EventTrace
	pf.DNDTrace = &DNDTrace
	pf.GoCompleteTrace = &golang.CompleteTrace
	pf.GoTypeTrace = &golang.TraceTypes
}

// Profile toggles profiling on / off
func (pf *PrefsDebug) Profile() {
	ProfileToggle()
}
