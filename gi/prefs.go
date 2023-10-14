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
	"log/slog"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"goki.dev/colors"
	"goki.dev/girl/paint"
	"goki.dev/girl/styles"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
	"goki.dev/pi/v2/langs/golang"
)

// Init performs overall initialization of the gogi system: loading prefs, etc
// -- automatically called when new window opened, but can be called before
// then if pref info needed.
func Init() {
	if Prefs.LogicalDPIScale == 0 {
		Prefs.Defaults()
		PrefsDet.Defaults()
		PrefsDbg.Connect()
		Prefs.Open()
		Prefs.Apply()
		goosi.InitScreenLogicalDPIFunc = Prefs.ApplyDPI // called when screens are initialized
		TheViewIFace.HiStyleInit()
		WinGeomMgr.NeedToReload() // gets time stamp associated with open, so it doesn't re-open
		WinGeomMgr.Open()
	}
}

// Preferences are the overall user preferences for GoGi, providing some basic
// customization -- in addition, most gui settings can be styled using
// CSS-style sheets under CustomStyle.  These prefs are saved and loaded from
// the GoGi user preferences directory -- see oswin/App for further info.
//
//gti:add
type Preferences struct {

	// the color theme
	Theme Themes

	// the primary color used to generate the color scheme
	Color color.RGBA

	// text highilighting style / theme
	HiStyle HiStyleName

	// the density (compactness) of content
	Density Densities

	// overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc
	LogicalDPIScale float32 `min:"0.1" step:"0.1"`

	// screen-specific preferences -- will override overall defaults if set
	ScreenPrefs map[string]ScreenPrefs

	// parameters controlling GUI behavior
	Params ParamPrefs `view:"inline"`

	// editor preferences -- for TextView etc
	Editor EditorPrefs `view:"inline"`

	// select the active keymap from list of available keymaps -- see Edit KeyMaps for editing / saving / loading that list
	KeyMap KeyMapName

	// if set, the current available set of key maps is saved to your preferences directory, and automatically loaded at startup -- this should be set if you are using custom key maps, but it may be safer to keep it <i>OFF</i> if you are <i>not</i> using custom key maps, so that you'll always have the latest compiled-in standard key maps with all the current key functions bound to standard key chords
	SaveKeyMaps bool

	// if set, the detailed preferences are saved and loaded at startup -- only
	SaveDetailed bool

	// a custom style sheet -- add a separate Props entry for each type of object, e.g., button, or class using .classname, or specific named element using #name -- all are case insensitive
	CustomStyles ki.Props

	// if true my custom styles override other styling (i.e., they come <i>last</i> in styling process -- otherwise they provide defaults that can be overridden by app-specific styling (i.e, they come first).
	CustomStylesOverride bool

	// default font family when otherwise not specified
	FontFamily FontName

	// default mono-spaced font family
	MonoFont FontName

	// extra font paths, beyond system defaults -- searched first
	FontPaths []string

	// user info -- partially filled-out automatically if empty / when prefs first created
	User User

	// favorite paths, shown in FileViewer and also editable there
	FavPaths FavPaths

	// column to sort by in FileView, and :up or :down for direction -- updated automatically via FileView
	FileViewSort string `view:"-"`

	// filename for saving / loading colors
	ColorFilename FileName `view:"-" ext:".json"`

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-"`
}

// Prefs are the overall preferences
var Prefs = Preferences{}

// OverridePrefsColor is whether to override the color specified in [Prefs.Color]
// with whatever the developer specifies, typically through [colors.SetSchemes].
// The intended usage is:
//
//	gi.OverridePrefsColor = true
//	colors.SetSchemes(colors.Green)
//
// It is recommended that you do not set this to give the user more control over
// their experience, but you can if you wish to enforce brand colors.
//
// The user preference color will always be overridden if it is the default value
// of Google Blue (#4285f4), so a more recommended option would be to set your
// own custom scheme but not OverridePrefsColor, giving you brand colors unless
// your user explicitly states a preference for a specific color.
var OverridePrefsColor = false

func (pf *Preferences) Defaults() {
	pf.Theme = ThemeAuto
	pf.Color = color.RGBA{66, 133, 244, 255} // Google Blue (#4285f4)
	pf.HiStyle = "emacs"                     // todo: "monokai" for dark mode.
	pf.Density = DensityMedium
	pf.LogicalDPIScale = 1.0
	pf.Params.Defaults()
	pf.Editor.Defaults()
	pf.FavPaths.SetToDefaults()
	pf.FontFamily = "Go"
	pf.MonoFont = "Go Mono"
	pf.KeyMap = DefaultKeyMap
	pf.UpdateUser()
}

// UpdateAll updates all open windows with current preferences -- triggers
// rebuild of default styles.
//
//gi:toolbar -icon refresh
func (pf *Preferences) UpdateAll() {
	goosi.ZoomFactor = 1 // reset so saved dpi is used
	pf.Apply()

	styles.RebuildDefaultStyles = true
	colors.FullCache = nil
	// for _, w := range AllRenderWins {  // no need and just messes stuff up!
	// 	w.SetSize(w.RenderWin.Size())
	// }
	// needs another pass through to get it right..
	// for _, w := range AllRenderWins {
	// 	w.FullReRender()
	// }
	styles.RebuildDefaultStyles = false
	// and another without rebuilding?  yep all are required
	// for _, w := range AllRenderWins {
	// 	w.FullReRender()
	// }
}

// PrefsFileName is the name of the preferences file in GoGi prefs directory
var PrefsFileName = "prefs.json"

// Open preferences from GoGi standard prefs directory
func (pf *Preferences) Open() error {
	pdir := goosi.TheApp.GoGiPrefsDir()
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
//
//gi:toolbar -sep-before
func (pf *Preferences) Save() error {
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, PrefsFileName)
	b, err := json.MarshalIndent(pf, "", "  ")
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	err = os.WriteFile(pnm, b, 0644)
	if err != nil {
		slog.Error(err.Error())
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

// TODO: need to handle auto theme and set things correctly

// LightMode sets the color theme to light mode. It automatically
// saves the preferences and updates all of the windows.
//
//gi:toolbar -sep-before
func (pf *Preferences) LightMode() {
	pf.Theme = ThemeLight
	colors.SetScheme(false)
	pf.Save()
	pf.UpdateAll()
}

// DarkMode sets the color theme to dark mode. It automatically
// saves the preferences and updates all of the windows.
//
//gi:toolbar
func (pf *Preferences) DarkMode() {
	pf.Theme = ThemeDark
	colors.SetScheme(true)
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
	// Google Blue (#4285f4) is the default value and thus indicates no user preference,
	// which means that we will always override the color, even without OverridePrefsColor
	if !OverridePrefsColor && pf.Color != (color.RGBA{66, 133, 244, 255}) {
		colors.SetSchemes(pf.Color)
	}
	switch pf.Theme {
	case ThemeLight:
		colors.SetScheme(false)
	case ThemeDark:
		colors.SetScheme(true)
	case ThemeAuto:
		colors.SetScheme(goosi.TheApp.IsDark())
	}
	if pf.HiStyle == "" {
		pf.HiStyle = "emacs" // todo: need light / dark versions
	}

	TheViewIFace.SetHiStyleDefault(pf.HiStyle)
	events.DoubleClickInterval = pf.Params.DoubleClickInterval
	events.ScrollWheelSpeed = pf.Params.ScrollWheelSpeed
	LocalMainMenu = pf.Params.LocalMainMenu

	if pf.KeyMap != "" {
		SetActiveKeyMapName(pf.KeyMap) // fills in missing pieces
	}
	if pf.SaveDetailed {
		PrefsDet.Apply()
	}
	if pf.FontPaths != nil {
		paths := append(pf.FontPaths, paint.FontPaths...)
		paint.FontLibrary.InitFontPaths(paths...)
	} else {
		paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	}
	pf.ApplyDPI()
}

// ApplyDPI updates the screen LogicalDPI values according to current
// preferences and zoom factor, and then updates all open windows as well.
func (pf *Preferences) ApplyDPI() {
	n := goosi.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := goosi.TheApp.Screen(i)
		if sc == nil {
			continue
		}
		if scp, ok := pf.ScreenPrefs[sc.Name]; ok {
			goosi.SetLogicalDPIScale(sc.Name, scp.LogicalDPIScale)
		}
		sc.UpdateLogicalDPI()
	}
	for _, w := range AllRenderWins {
		w.GoosiWin.SetLogicalDPI(w.GoosiWin.Screen().LogicalDPI)
	}
}

// SaveZoom saves the current LogicalDPI scaling, either as the overall
// default or specific to the current screen.
//
//gi:toolbar -sep-before -icon zoom_in
func (pf *Preferences) SaveZoom(forCurrentScreen bool) {
	sc := goosi.TheApp.Screen(0)
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

// ScreenInfo returns screen info for all screens on the console.
//
//gi:toolbar -icon info
func (pf *Preferences) ScreenInfo() string {
	ns := goosi.TheApp.NScreens()
	scinfo := ""
	for i := 0; i < ns; i++ {
		sc := goosi.TheApp.Screen(i)
		if i > 0 {
			scinfo += "<br><br>\n"
		}
		scinfo += fmt.Sprintf("Screen number: %v Name: %v<br>\n    Geom: %v, DevPixRatio: %v<br>\n    Pixels: %v, Physical size: %v mm<br>\n    Logical DPI: %v, Physical DPI: %v, Logical DPI scale: %v<br>\n    Depth: %v, Refresh rate: %v<br>\n    Orientation: %v, Native orientation: %v, Primary orientation: %v<br>\n", i, sc.Name, sc.Geometry, sc.DevicePixelRatio, sc.PixSize, sc.PhysicalSize, sc.LogicalDPI, sc.PhysicalDPI, sc.LogicalDPI/sc.PhysicalDPI, sc.Depth, sc.RefreshRate, sc.Orientation, sc.NativeOrientation, sc.PrimaryOrientation)
	}
	return scinfo
}

// VersionInfo returns GoGi version information
//
//gi:toolbar -icon info
func (pf *Preferences) VersionInfo() string {
	vinfo := Version + " date: " + VersionDate + " UTC; git commit-1: " + GitCommit
	return vinfo
}

// DeleteSavedRenderWinGeoms deletes the file that saves the position and size of
// each window, by screen, and clear current in-memory cache.  You shouldn't
// need to use this but sometimes useful for testing.
func (pf *Preferences) DeleteSavedRenderWinGeoms() {
	WinGeomMgr.DeleteAll()
}

// EditKeyMaps opens the KeyMapsView editor to create new keymaps / save /
// load from other files, etc.  Current avail keymaps are saved and loaded
// with preferences automatically.
//
//gi:toolbar -sep-before -icon keyboard
func (pf *Preferences) EditKeyMaps() {
	pf.SaveKeyMaps = true
	pf.Changed = true
	TheViewIFace.KeyMapsView(&AvailKeyMaps)
}

// EditHiStyles opens the HiStyleView editor to customize highlighting styles
//
//gi:toolbar -icon ink_highlighter
func (pf *Preferences) EditHiStyles() {
	TheViewIFace.HiStylesView(false) // false = custom
}

// EditDetailed opens the PrefsDetView editor to edit detailed params
//
//gi:toolbar -icon description
func (pf *Preferences) EditDetailed() {
	pf.SaveDetailed = true
	pf.Changed = true
	TheViewIFace.PrefsDetView(&PrefsDet)
}

// EditDebug opens the PrefsDbgView editor to edit debugging params
//
//gi:toolbar -icon bug_report
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

// PrefFontFamily returns the default FontFamily
func (pf *Preferences) PrefFontFamily() string {
	// TODO: where should this go?
	return string(pf.FontFamily)
}

// PreferencesProps define the Toolbar and MenuBar for StructView, e.g., giv.PrefsView
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
				"updtfunc": func(pfi any, bt *Button) {
					pf := pfi.(*Preferences)
					bt.SetEnabledState(pf.Changed)
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
			{"DeleteSavedRenderWinGeoms", ki.Props{
				"confirm": true,
				"desc":    "Are you <i>sure</i>?  This deletes the file that saves the position and size of each window, by screen, and clear current in-memory cache.  You shouldn't generally need to do this but sometimes it is useful for testing or windows are showing up in bad places that you can't recover from.",
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close RenderWin", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"RenderWin", "RenderWins"},
	},
	"Toolbar": ki.PropSlice{
		{"UpdateAll", ki.Props{
			"desc": "Updates all open windows with current preferences -- triggers rebuild of default styles.",
			"icon": icons.Refresh,
		}},
		{"sep-file", ki.BlankProp{}},
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs.json file, which is auto-loaded at startup.",
			"icon": icons.Save,
			"updtfunc": func(pfi any, bt *Button) {
				pf := pfi.(*Preferences)
				bt.SetEnabledStateUpdt(pf.Changed)
			},
		}},
		{"sep-color", ki.BlankProp{}},
		{"LightMode", ki.Props{
			"desc": "Set color mode to Light mode as defined in ColorSchemes -- automatically does Save and UpdateAll ",
			"icon": icons.LightMode,
		}},
		{"DarkMode", ki.Props{
			"desc": "Set color mode to Dark mode as defined in ColorSchemes -- automatically does Save and UpdateAll",
			"icon": icons.DarkMode,
		}},
		{"sep-scrn", ki.BlankProp{}},
		{"SaveZoom", ki.Props{
			"icon": icons.ZoomIn,
			"desc": "Save current zoom magnification factor, either for all screens or for the current screen only",
			"Args": ki.PropSlice{
				{"For Current Screen Only?", ki.Props{
					"desc":    "click this to save zoom specifically for current screen",
					"default": true,
				}},
			},
		}},
		{"ScreenInfo", ki.Props{
			"desc":        "shows parameters about all the active screens",
			"icon":        icons.Info,
			"show-return": true,
		}},
		{"VersionInfo", ki.Props{
			"desc":        "shows current GoGi version information",
			"icon":        icons.Info,
			"show-return": true,
		}},
		{"sep-key", ki.BlankProp{}},
		{"EditKeyMaps", ki.Props{
			"icon": icons.Keyboard,
			"desc": "opens the KeyMapsView editor to create new keymaps / save / load from other files, etc.  Current keymaps are saved and loaded with preferences automatically if SaveKeyMaps is clicked (will be turned on automatically if you open this editor).",
		}},
		{"EditHiStyles", ki.Props{
			"icon": icons.InkHighlighter,
			"desc": "opens the HiStylesView editor of highlighting styles.",
		}},
		{"EditDetailed", ki.Props{
			"icon": icons.Description,
			"desc": "opens the PrefsDetView editor to edit detailed params that are not typically user-modified, but can be if you really care..  Turns on the SaveDetailed flag so these will be saved and loaded automatically -- can toggle that back off if you don't actually want to.",
		}},
		{"EditDebug", ki.Props{
			"icon": icons.BugReport,
			"desc": "Opens the PrefsDbgView editor to control debugging parameters. These are not saved -- only set dynamically during running.",
		}},
	},
}

// Densities is an enum representing the different
// density options in user preferences
type Densities int32 //enums:enum -trimprefix Density

const (
	// DensityCompact represents a compact density
	// with minimal whitespace
	DensityCompact Densities = iota
	// DensityMedium represents a medium density
	// with medium whitespace
	DensityMedium
	// DensitySpread represents a spread-out density
	// with a lot of whitespace
	DensitySpread
)

// DensityMul returns a multiplier centered
// around 1 representing the density set in the preferences.
// It should be used for determining padding and margin values.
func (pf *Preferences) DensityMul() float32 {
	switch pf.Density {
	case DensityCompact:
		return 0.5
	case DensityMedium:
		return 1
	case DensitySpread:
		return 1.5
	}
	log.Println("got invalid preferences density value", pf.Density)
	return 1
}

//////////////////////////////////////////////////////////////////
//  ParamPrefs

// ScreenPrefs are the per-screen preferences -- see oswin/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences.
//
//gti:add
type ScreenPrefs struct {

	// overall scaling factor for Logical DPI as a multiplier on Physical DPI -- smaller numbers produce smaller font sizes etc.  Actual Logical DPI is enforced to be a multiple of 6, so the precise number here isn't critical -- rounding to 2 digits is more than sufficient.
	LogicalDPIScale float32 `min:"0.1" step:"0.1"`
}

// ParamPrefs contains misc parameters controlling GUI behavior.
//
//gti:add
type ParamPrefs struct {

	// the maximum time interval in msec between button press events to count as a double-click
	DoubleClickInterval time.Duration `min:"100" step:"50"`

	// how fast the scroll wheel moves -- typically pixels per wheel step but units can be arbitrary.  It is generally impossible to standardize speed and variable across devices, and we don't have access to the system settings, so unfortunately you have to set it here.
	ScrollWheelSpeed float32 `min:"0.01" step:"1"`

	// controls whether the main menu is displayed locally at top of each window, in addition to global menu at the top of the screen.  Mac native apps do not do this, but OTOH it makes things more consistent with other platforms, and with larger screens, it can be convenient to have access to all the menu items right there.
	LocalMainMenu bool

	// only support closing the currently selected active tab; if this is set to true, pressing the close button on other tabs will take you to that tab, from which you can close it
	OnlyCloseActiveTab bool `def:"false"`

	// the amount that alternating rows and columns are highlighted when showing tabular data (set to 0 to disable zebra striping)
	ZebraStripeWeight float32 `def:"0" min:"0" max:"100" step:"1"`

	// the limit of file size, above which user will be prompted before opening / copying, etc.
	BigFileSize int `def:"10000000"`

	// maximum number of saved paths to save in FileView
	SavedPathsMax int

	// turn on smoothing in 3D rendering -- this should be on by default but if you get an error telling you to turn it off, then do so (because your hardware can't handle it)
	Smooth3D bool
}

func (pf *ParamPrefs) Defaults() {
	pf.DoubleClickInterval = 500 * time.Millisecond
	pf.ScrollWheelSpeed = 20
	pf.LocalMainMenu = true // much better
	pf.OnlyCloseActiveTab = false
	pf.ZebraStripeWeight = 0
	pf.BigFileSize = 10000000
	pf.SavedPathsMax = 50
	pf.Smooth3D = true
}

// User basic user information that might be needed for different apps
//
//gti:add
type User struct {
	user.User

	// default email address -- e.g., for recording changes in a version control system
	Email string
}

//////////////////////////////////////////////////////////////////
//  EditorPrefs

// EditorPrefs contains editor preferences.  It can also be set
// from ki.Props style properties.
//
//gti:add
type EditorPrefs struct {

	// size of a tab, in chars -- also determines indent level for space indent
	TabSize int `xml:"tab-size"`

	// use spaces for indentation, otherwise tabs
	SpaceIndent bool `xml:"space-indent"`

	// wrap lines at word boundaries -- otherwise long lines scroll off the end
	WordWrap bool `xml:"word-wrap"`

	// show line numbers
	LineNos bool `xml:"line-nos"`

	// use the completion system to suggest options while typing
	Completion bool `xml:"completion"`

	// suggest corrections for unknown words while typing
	SpellCorrect bool `xml:"spell-correct"`

	// automatically indent lines when enter, tab, }, etc pressed
	AutoIndent bool `xml:"auto-indent"`

	// use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo
	EmacsUndo bool `xml:"emacs-undo"`

	// colorize the background according to nesting depth
	DepthColor bool `xml:"depth-color"`
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

//////////////////////////////////////////////////////////////////
//  FavoritePaths

// FavPathItem represents one item in a favorite path list, for display of
// favorites.  Is an ordered list instead of a map because user can organize
// in order
//
//gti:add
type FavPathItem struct {

	// icon for item
	Ic icons.Icon

	// name of the favorite item
	Name string `width:"20"`

	//
	Path string `tableview:"-select"`
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
	{icons.Home, "home", "~"},
	{icons.DesktopMac, "Desktop", "~/Desktop"},
	{icons.LabProfile, "Documents", "~/Documents"},
	{icons.Download, "Downloads", "~/Downloads"},
	{icons.Computer, "root", "/"},
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
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.SaveJSON(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := goosi.TheApp.GoGiPrefsDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.OpenJSON(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

//////////////////////////////////////////////////////////////////
//  PrefsDetailed

// TODO: make all of the MSec things time.Duration

// PrefsDetailed are more detailed params not usually customized, but
// available for those who really care..
//
//gti:add
type PrefsDetailed struct {

	// the maximum height of any menu popup panel in units of font height -- scroll bars are enforced beyond that size.
	MenuMaxHeight int `def:"30" min:"5" step:"1"`

	// the number of milliseconds to wait before initiating a regular mouse drag event (as opposed to a basic events.Press)
	DragStartTime time.Duration `def:"50" min:"5" max:"1000" step:"5"`

	// the number of pixels that must be moved before initiating a regular mouse drag event (as opposed to a basic events.Press)
	DragStartDist int `def:"4" min:"0" max:"100" step:"1"`

	// the number of milliseconds to wait before initiating a drag-n-drop event -- gotta drag it like you mean it
	SlideStartTime time.Duration `def:"200" min:"5" max:"1000" step:"5"`

	// the number of pixels that must be moved before initiating a drag-n-drop event -- gotta drag it like you mean it
	SlideStartDist int `def:"20" min:"0" max:"100" step:"1"`

	// the number of milliseconds to wait before initiating a hover event (e.g., for opening a tooltip)
	LongHoverTime time.Duration `def:"500" min:"10" max:"10000" step:"10"`

	// the maximum number of pixels that mouse can move and still register a Hover event
	LongHoverStopDist int `def:"50" min:"0" max:"1000" step:"1"`

	// the number of milliseconds to wait before offering completions
	CompleteWaitMSec int `def:"0" min:"0" max:"10000" step:"10"`

	// the maximum number of completions offered in popup
	CompleteMaxItems int `def:"25" min:"5" step:"1"`

	// time interval for cursor blinking on and off -- set to 0 to disable blinking
	CursorBlinkTime time.Duration `def:"500" min:"0" max:"1000" step:"5"`

	// is amount of time to wait (in Milliseconds) before trying to autoscroll again
	LayoutAutoScrollDelayMSec int `def:"25" min:"1" step:"5"`

	// number of steps to take in PageUp / Down events in terms of number of items
	LayoutPageSteps int `def:"10" min:"1" step:"1"`

	// the number of milliseconds between keypresses to combine characters into name to search for within layout -- starts over after this delay
	LayoutFocusNameTimeoutMSec int `def:"500" min:"0" max:"5000" step:"20"`

	// the number of milliseconds since last focus name event to allow tab to focus on next element with same name.
	LayoutFocusNameTabMSec int `def:"2000" min:"10" max:"10000" step:"100"`

	// open dialogs in separate windows -- else do as popups in main window
	DialogsSepRenderWin bool `def:"true"`

	// Maximum amount of clipboard history to retain
	TextViewClipHistMax int `def:"100" min:"0" max:"1000" step:"5"`

	// maximum number of lines to look for matching scope syntax (parens, brackets)
	TextBufMaxScopeLines int `def:"100" min:"10" step:"10"`

	// text buffer max lines to use diff-based revert to more quickly update e.g., after file has been reformatted
	TextBufDiffRevertLines int `def:"10000" min:"0" step:"1000"`

	// text buffer max diffs to use diff-based revert to more quickly update e.g., after file has been reformatted -- if too many differences, just revert
	TextBufDiffRevertDiffs int `def:"20" min:"0" step:"1"`

	// number of milliseconds to wait before starting a new background markup process, after text changes within a single line (always does after line insertion / deletion)
	TextBufMarkupDelayMSec int `def:"1000" min:"100" step:"100"`

	// the number of map elements at or below which an inline representation of the map will be presented -- more convenient for small #'s of props
	MapInlineLen int `def:"3" min:"2" step:"1"`

	// the number of elemental struct fields at or below which an inline representation of the struct will be presented -- more convenient for small structs
	StructInlineLen int `def:"6" min:"2" step:"1"`

	// the number of slice elements below which inline will be used
	SliceInlineLen int `def:"6" min:"2" step:"1"`

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-"`
}

// PrefsDet are the overall detailed preferences
var PrefsDet = PrefsDetailed{}

// PrefsDetailedFileName is the name of the detailed preferences file in GoGi prefs directory
var PrefsDetailedFileName = "prefs_det.json"

// Open detailed preferences from GoGi standard prefs directory
func (pf *PrefsDetailed) Open() error {
	pdir := goosi.TheApp.GoGiPrefsDir()
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
	pdir := goosi.TheApp.GoGiPrefsDir()
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
	pf.DragStartTime = DragStartTime
	pf.DragStartDist = DragStartDist
	pf.SlideStartTime = SlideStartTime
	pf.SlideStartDist = SlideStartDist
	pf.LongHoverTime = LongHoverTime
	pf.LongHoverStopDist = LongHoverStopDist
	pf.CompleteWaitMSec = CompleteWaitMSec
	pf.CompleteMaxItems = CompleteMaxItems
	pf.CursorBlinkTime = CursorBlinkTime
	pf.LayoutAutoScrollDelayMSec = LayoutAutoScrollDelayMSec
	pf.LayoutPageSteps = LayoutPageSteps
	pf.LayoutFocusNameTimeoutMSec = LayoutFocusNameTimeoutMSec
	pf.LayoutFocusNameTabMSec = LayoutFocusNameTabMSec
	pf.MenuMaxHeight = MenuMaxHeight
	// TheViewIFace.PrefsDetDefaults(pf) // todo:
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
	DragStartTime = pf.DragStartTime
	DragStartDist = pf.DragStartDist
	SlideStartTime = pf.SlideStartTime
	SlideStartDist = pf.SlideStartDist
	LongHoverTime = pf.LongHoverTime
	LongHoverStopDist = pf.LongHoverStopDist
	CompleteWaitMSec = pf.CompleteWaitMSec
	CompleteMaxItems = pf.CompleteMaxItems
	CursorBlinkTime = pf.CursorBlinkTime
	LayoutFocusNameTimeoutMSec = pf.LayoutFocusNameTimeoutMSec
	LayoutFocusNameTabMSec = pf.LayoutFocusNameTabMSec
	MenuMaxHeight = pf.MenuMaxHeight
	// TheViewIFace.PrefsDetApply(pf) // TODO: add this once we have giv working
	// in giv:
	// TextViewClipHistMax = pf.TextViewClipHistMax
	// TextBuf*
	// MapInlineLen
	// StructInlineLen
	// SliceInlineLen
}

// PrefsDetailedProps define the Toolbar and MenuBar for StructView, e.g., giv.PrefsDetView
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
				"updtfunc": func(pfi any, bt *Button) {
					pf := pfi.(*PrefsDetailed)
					bt.SetEnabledState(pf.Changed)
				},
			}},
			{"Close RenderWin", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste"},
		{"RenderWin", "RenderWins"},
	},
	"Toolbar": ki.PropSlice{
		{"Apply", ki.Props{
			"desc": "Apply parameters to affect actual behavior.",
			"icon": icons.Refresh,
		}},
		{"sep-file", ki.BlankProp{}},
		{"Save", ki.Props{
			"desc": "Saves current preferences to standard prefs_det.json file, which is auto-loaded at startup.",
			"icon": icons.Save,
			"updtfunc": func(pfi any, bt *Button) {
				pf := pfi.(*PrefsDetailed)
				bt.SetEnabledStateUpdt(pf.Changed)
			},
		}},
	},
}

//////////////////////////////////////////////////////////////////
//  PrefsDebug

// StrucdtViewIfDebug is a debug flag for getting error messages on
// viewif struct tag directives in the giv.StructView.
var StructViewIfDebug = false

// PrefsDebug are debugging params
type PrefsDebug struct {

	// reports trace of updates that trigger re-rendering (printfs to stdout)
	UpdateTrace *bool

	// reports trace of the nodes rendering (printfs to stdout)
	RenderTrace *bool

	// reports trace of all layouts (printfs to stdout)
	LayoutTrace *bool

	// reports trace of window events (printfs to stdout)
	WinEventTrace *bool

	// reports the stack trace leading up to win publish events which are expensive -- wrap multiple updates in UpdateStart / End to prevent
	WinRenderTrace *bool

	// WinGeomTrace records window geometry saving / loading functions
	WinGeomTrace *bool

	// reports trace of keyboard events (printfs to stdout)
	KeyEventTrace *bool

	// reports trace of event handling (printfs to stdout)
	EventTrace *bool

	// reports trace of DND events handling
	DNDTrace *bool

	// reports trace of Go language completion & lookup process
	GoCompleteTrace *bool

	// reports trace of Go language type parsing and inference process
	GoTypeTrace *bool

	// reports errors for viewif directives in struct field tags, for giv.StructView
	StructViewIfDebug *bool

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" xml:"-"`
}

// PrefsDbg are the overall debugging preferences
var PrefsDbg = PrefsDebug{}

// PrefsDebugProps define the Toolbar and MenuBar for StructView, e.g., giv.PrefsDbgView
var PrefsDebugProps = ki.Props{
	"Toolbar": ki.PropSlice{
		{"Profile", ki.Props{
			"desc": "Toggle profiling of program on or off -- does both targeted and global CPU and Memory profiling.",
			"icon": icons.LabProfile,
		}},
	},
}

// Connect connects debug fields with actual variables controlling debugging
func (pf *PrefsDebug) Connect() {
	pf.UpdateTrace = &UpdateTrace
	pf.RenderTrace = &RenderTrace
	pf.LayoutTrace = &LayoutTrace
	pf.WinEventTrace = &WinEventTrace
	pf.WinRenderTrace = &WinRenderTrace
	pf.WinGeomTrace = &WinGeomTrace
	pf.KeyEventTrace = &KeyEventTrace
	pf.EventTrace = &EventTrace
	pf.GoCompleteTrace = &golang.CompleteTrace
	pf.GoTypeTrace = &golang.TraceTypes
	pf.StructViewIfDebug = &StructViewIfDebug
}

// Profile toggles profiling on / off
func (pf *PrefsDebug) Profile() {
	// ProfileToggle()
}
