// Copyright (c) 2018, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image/color"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"goki.dev/colors"
	"goki.dev/colors/gradient"
	"goki.dev/colors/matcolor"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/paint"
	"goki.dev/glop/option"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/grows/jsons"
	"goki.dev/grows/tomls"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ordmap"
	"goki.dev/pi/v2/langs/golang"
)

// AllSettings is a global ordered map containing all of the user [Settings]
// that the user will see in the settings window. It contains the base Goki
// settings by default and should be modified by other apps to add their
// app settings.
var AllSettings = ordmap.Make([]ordmap.KeyVal[string, Settings]{
	{"General", GeneralSettings},
	{"Devices", DeviceSettings},
	{"Debugging", DebugSettings},
})

// Settings is the interface that describes the functionality common to all settings data types.
type Settings interface {

	// Filename returns the filename/filepath at which the settings are stored relative to [DataDir].
	Filename() string

	// Defaults sets the default values for all of the settings.
	Defaults()

	// Apply does anything necessary to apply the settings to the app.
	Apply()
}

// SettingsBase contains base settings logic that other settings data types can extend.
type SettingsBase struct {
	// File is the filename/filepath at which the settings are stored relative to [DataDir].
	File string `view:"-"`
}

// Filename returns the full filename/filepath at which the settings are stored.
func (sb *SettingsBase) Filename() string {
	return sb.File
}

// Defaults does nothing by default and can be extended by other settings data types.
func (sb *SettingsBase) Defaults() {}

// Apply does nothing by default and can be extended by other settings data types.
func (sb *SettingsBase) Apply() {}

// OpenSettings opens the given settings from their [Settings.Filename].
// The settings must be encoded in TOML.
func OpenSettings(se Settings) error {
	return tomls.Open(se, filepath.Join(DataDir(), se.Filename()))
}

// SaveSettings saves the given settings to their [Settings.Filename].
// It encodes the settings in TOML.
func SaveSettings(se Settings) error {
	return tomls.Save(se, filepath.Join(DataDir(), se.Filename()))
}

// ResetSettings resets the given settings to their default values.
func ResetSettings(se Settings) error {
	err := os.Remove(filepath.Join(DataDir(), se.Filename()))
	if err != nil {
		return err
	}
	se.Defaults()
	return nil
}

// LoadSettings sets the defaults of, opens, and applies the given settings.
func LoadSettings(se Settings) error {
	se.Defaults()
	err := OpenSettings(se)
	if err != nil {
		return err
	}
	se.Apply()
	return nil
}

// LoadAllSettings sets the defaults of, opens, and applies [AllSettings].
func LoadAllSettings() error {
	for _, kv := range AllSettings.Order {
		err := LoadSettings(kv.Val)
		if err != nil {
			return err
		}
	}
	return nil
}

// Init performs the overall initialization of the Goki system by loading
// settings. It is automatically called when a new window opened, but can
// be called before then if certain settings info needed.
func Init() {
	goosi.InitScreenLogicalDPIFunc = GeneralSettings.ApplyDPI // called when screens are initialized
	grr.Log(LoadAllSettings())
	WinGeomMgr.NeedToReload() // gets time stamp associated with open, so it doesn't re-open
	WinGeomMgr.Open()
}

// GeneralSettings are the currently active global Goki general settings.
var GeneralSettings = &GeneralSettingsData{
	SettingsBase: SettingsBase{
		File: filepath.Join("goki", "general-settings.toml"),
	},
}

// GeneralSettingsData is the data type for the general Goki settings.
// The global current instance is stored as [GeneralSettings].
type GeneralSettingsData struct { //gti:add
	SettingsBase

	// the color theme
	Theme Themes

	// the primary color used to generate the color scheme
	Color color.RGBA

	// overall zoom factor as a percentage of the default zoom
	Zoom float32 `def:"100" min:"10" max:"1000" step:"10" format:"%g%%"`

	// the overall spacing factor as a percentage of the default amount of spacing
	// (higher numbers lead to more space and lower numbers lead to higher density)
	Spacing float32 `def:"100" min:"10" max:"1000" step:"10" format:"%g%%"`

	// the overall font size factor applied to all text as a percentage
	// of the default font size (higher numbers lead to larger text)
	FontSize float32 `def:"100" min:"10" max:"1000" step:"10" format:"%g%%"`

	// screen-specific preferences -- will override overall defaults if set
	ScreenPrefs map[string]ScreenPrefs

	// text highlighting style / theme
	HiStyle HiStyleName

	// whether to use a 24-hour clock (instead of AM and PM)
	Clock24 bool `label:"24-hour clock"`

	// parameters controlling GUI behavior
	Params ParamPrefs

	// editor preferences -- for TextEditor etc
	Editor EditorPrefs

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
	ColorFilename FileName `view:"-" ext:".toml"`

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" toml:"-" xml:"-"`
}

// OverrideSettingsColor is whether to override the color specified in [Prefs.Color]
// with whatever the developer specifies, typically through [colors.SetSchemes].
// The intended usage is:
//
//	gi.OverrideSettingsColor = true
//	colors.SetSchemes(colors.Green)
//
// It is recommended that you do not set this to give the user more control over
// their experience, but you can if you wish to enforce brand colors.
//
// The user preference color will always be overridden if it is the default value
// of Google Blue (#4285f4), so a more recommended option would be to set your
// own custom scheme but not OverrideSettingsColor, giving you brand colors unless
// your user explicitly states a preference for a specific color.
var OverrideSettingsColor = false

func (pf *GeneralSettingsData) Defaults() {
	pf.Theme = ThemeAuto
	pf.Color = color.RGBA{66, 133, 244, 255} // Google Blue (#4285f4)
	pf.HiStyle = "emacs"                     // todo: "monokai" for dark mode.
	pf.Zoom = 100
	pf.Spacing = 100
	pf.FontSize = 100
	pf.Params.Defaults()
	pf.Editor.Defaults()
	pf.FavPaths.SetToDefaults()
	pf.FontFamily = "Roboto"
	pf.MonoFont = "Roboto Mono"
	pf.UpdateUser()
}

// UpdateAll updates all windows and triggers a full render rebuild.
// It is typically called when user settings are changed.
func UpdateAll() { //gti:add
	gradient.Cache = nil // the cache is invalid now
	for _, w := range AllRenderWins {
		rctx := w.MainStageMgr.RenderCtx
		rctx.LogicalDPI = w.LogicalDPI()
		rctx.SetFlag(true, RenderRebuild) // trigger full rebuild
	}
}

// Apply applies the settings.
func (pf *GeneralSettingsData) Apply() { //gti:add
	np := len(pf.FavPaths)
	for i := 0; i < np; i++ {
		if pf.FavPaths[i].Ic == "" {
			pf.FavPaths[i].Ic = "folder"
		}
	}
	// Google Blue (#4285f4) is the default value and thus indicates no user preference,
	// which means that we will always override the color, even without OverridePrefsColor
	if !OverrideSettingsColor && pf.Color != (color.RGBA{66, 133, 244, 255}) {
		colors.SetSchemes(pf.Color)
	}
	// TODO(kai): figure out transparency approach
	// colors.Schemes.Dark.Background.A = 250
	// colors.Schemes.Light.Background.A = 250
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

	if TheViewIFace != nil {
		TheViewIFace.SetHiStyleDefault(pf.HiStyle)
	}
	LocalMainMenu = pf.Params.LocalMainMenu

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
func (pf *GeneralSettingsData) ApplyDPI() {
	// zoom is percentage, but LogicalDPIScale is multiplier
	goosi.LogicalDPIScale = pf.Zoom / 100
	// fmt.Println("goosi ldpi:", goosi.LogicalDPIScale)
	n := goosi.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := goosi.TheApp.Screen(i)
		if sc == nil {
			continue
		}
		if scp, ok := pf.ScreenPrefs[sc.Name]; ok {
			// zoom is percentage, but LogicalDPIScale is multiplier
			goosi.SetLogicalDPIScale(sc.Name, scp.Zoom/100)
		}
		sc.UpdateLogicalDPI()
	}
	for _, w := range AllRenderWins {
		w.GoosiWin.SetLogicalDPI(w.GoosiWin.Screen().LogicalDPI)
		// this isn't DPI-related, but this is the most efficient place to do it
		w.GoosiWin.SetTitleBarIsDark(matcolor.SchemeIsDark)
	}
}

/*
// SaveZoom saves the current LogicalDPI scaling, either as the overall
// default or specific to the current screen.
//   - forCurrentScreen: if true, saves only for current screen
func (pf *GeneralSettingsData) SaveZoom(forCurrentScreen bool) { //gti:add
	goosi.ZoomFactor = 1 // reset -- otherwise has 2x effect
	sc := goosi.TheApp.Screen(0)
	if forCurrentScreen {
		sp, ok := pf.ScreenPrefs[sc.Name]
		if !ok {
			sp = ScreenPrefs{}
		}
		sp.Zoom = mat32.Truncate(100*sc.LogicalDPI/sc.PhysicalDPI, 2)
		if pf.ScreenPrefs == nil {
			pf.ScreenPrefs = make(map[string]ScreenPrefs)
		}
		pf.ScreenPrefs[sc.Name] = sp
	} else {
		pf.Zoom = mat32.Truncate(100*sc.LogicalDPI/sc.PhysicalDPI, 2)
	}
	grr.Log(pf.Save())
}
*/

// ScreenInfo returns screen info for all screens on the device
func (pf *GeneralSettingsData) ScreenInfo() []*goosi.Screen { //gti:add
	ns := goosi.TheApp.NScreens()
	res := make([]*goosi.Screen, ns)
	for i := 0; i < ns; i++ {
		res[i] = goosi.TheApp.Screen(i)
	}
	return res
}

// VersionInfo returns GoGi version information
func (pf *GeneralSettingsData) VersionInfo() string { //gti:add
	vinfo := "Version: " + Version + "\nDate: " + VersionDate + " UTC\nGit commit: " + GitCommit
	return vinfo
}

// DeleteSavedWindowGeoms deletes the file that saves the position and size of
// each window, by screen, and clear current in-memory cache. You shouldn't generally
// need to do this, but sometimes it is useful for testing or windows that are
// showing up in bad places that you can't recover from.
func (pf *GeneralSettingsData) DeleteSavedWindowGeoms() { //gti:add
	WinGeomMgr.DeleteAll()
}

// EditHiStyles opens the HiStyleView editor to customize highlighting styles
func (pf *GeneralSettingsData) EditHiStyles() { //gti:add
	TheViewIFace.HiStylesView(false) // false = custom
}

// UpdateUser gets the user info from the OS
func (pf *GeneralSettingsData) UpdateUser() {
	usr, err := user.Current()
	if err == nil {
		pf.User.User = *usr
	}
}

// PrefFontFamily returns the default FontFamily
func (pf *GeneralSettingsData) PrefFontFamily() string {
	// TODO: where should this go?
	return string(pf.FontFamily)
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

// DensityMul returns an enum value representing the type
// of density that the user has selected, based on a set of
// fixed breakpoints.
func (pf *GeneralSettingsData) DensityType() Densities {
	switch {
	case pf.Spacing < 50:
		return DensityCompact
	case pf.Spacing > 150:
		return DensitySpread
	default:
		return DensityMedium
	}
}

// TimeFormat returns the Go time format layout string that should
// be used for displaying times to the user, based on the value of
// [Prefs.Clock24].
func (pf *GeneralSettingsData) TimeFormat() string {
	if pf.Clock24 {
		return "15:04"
	}
	return "3:04 PM"
}

// DeviceSettings are the global device settings.
var DeviceSettings = &DeviceSettingsData{
	SettingsBase: SettingsBase{
		File: filepath.Join("goki", "device-settings.toml"),
	},
}

// DeviceSettingsData is the data type for the device settings.
type DeviceSettingsData struct {
	SettingsBase

	// The keyboard shortcut map to use
	KeyMap keyfun.MapName

	// The keyboard shortcut maps available as options for Key map.
	// If you do not want to have custom key maps, you should leave
	// this unset so that you always have the latest standard key maps.
	KeyMaps option.Option[keyfun.Maps]

	// The maximum time interval between button press events to count as a double-click
	DoubleClickInterval time.Duration `min:"100" step:"50"`

	// How fast the scroll wheel moves, which is typically pixels per wheel step
	// but units can be arbitrary. It is generally impossible to standardize speed
	// and variable across devices, and we don't have access to the system settings,
	// so unfortunately you have to set it here.
	ScrollWheelSpeed float32 `min:"0.01" step:"1"`

	// The amount of time to wait before initiating a regular slide event
	// (as opposed to a basic press event)
	SlideStartTime time.Duration `def:"50" min:"5" max:"1000" step:"5"`

	// The number of pixels that must be moved before initiating a regular
	// slide event (as opposed to a basic press event)
	SlideStartDistance int `def:"4" min:"0" max:"100" step:"1"`

	// The amount of time to wait before initiating a drag-n-drop event
	DragStartTime time.Duration `def:"200" min:"5" max:"1000" step:"5"`

	// The number of pixels that must be moved before initiating a drag-n-drop event
	DragStartDistance int `def:"20" min:"0" max:"100" step:"1"`

	// The amount of time to wait before initiating a long hover event (e.g., for opening a tooltip)
	LongHoverTime time.Duration `def:"500" min:"10" max:"10000" step:"10"`

	// The maximum number of pixels that mouse can move and still register a long hover event
	LongHoverStopDistance int `def:"50" min:"0" max:"1000" step:"1"`

	// The amount of time to wait before initiating a long press event (e.g., for opening a tooltip)
	LongPressTime time.Duration `def:"500" min:"10" max:"10000" step:"10"`

	// The maximum number of pixels that mouse/finger can move and still register a long press event
	LongPressStopDistance int `def:"50" min:"0" max:"1000" step:"1"`
}

func (ds *DeviceSettingsData) Defaults() {
	ds.KeyMap = keyfun.DefaultMap
	ds.KeyMaps.Value = keyfun.AvailMaps

	ds.DoubleClickInterval = 500 * time.Millisecond
	ds.ScrollWheelSpeed = 20

	ds.DragStartTime = DragStartTime
	ds.DragStartDistance = DragStartDistance
	ds.SlideStartTime = SlideStartTime
	ds.SlideStartDistance = SlideStartDistance
	ds.LongHoverTime = LongHoverTime
	ds.LongHoverStopDistance = LongHoverStopDistance
	ds.LongPressTime = LongPressTime
	ds.LongPressStopDistance = LongPressStopDistance
}

func (ds *DeviceSettingsData) Apply() {
	if ds.KeyMaps.Valid {
		keyfun.AvailMaps = ds.KeyMaps.Value
	}
	if ds.KeyMap != "" {
		keyfun.SetActiveMapName(ds.KeyMap)
	}

	events.DoubleClickInterval = ds.DoubleClickInterval
	events.ScrollWheelSpeed = ds.ScrollWheelSpeed

	DragStartTime = ds.DragStartTime
	DragStartDistance = ds.DragStartDistance
	SlideStartTime = ds.SlideStartTime
	SlideStartDistance = ds.SlideStartDistance
	LongHoverTime = ds.LongHoverTime
	LongHoverStopDistance = ds.LongHoverStopDistance
	LongPressTime = ds.LongPressTime
	LongPressStopDistance = ds.LongPressStopDistance
}

//////////////////////////////////////////////////////////////////
//  ParamPrefs

// ScreenPrefs are the per-screen preferences -- see goosi/App/Screen() for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global preferences.
type ScreenPrefs struct { //gti:add

	// overall zoom factor as a percentage of the default zoom
	Zoom float32 `def:"100" min:"10" max:"1000" step:"10"`
}

// ParamPrefs contains misc parameters controlling GUI behavior.
type ParamPrefs struct { //gti:add

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
	pf.LocalMainMenu = true // much better
	pf.OnlyCloseActiveTab = false
	pf.ZebraStripeWeight = 0
	pf.BigFileSize = 10000000
	pf.SavedPathsMax = 50
	pf.Smooth3D = true
}

// User basic user information that might be needed for different apps
type User struct { //gti:add
	user.User

	// default email address -- e.g., for recording changes in a version control system
	Email string
}

//////////////////////////////////////////////////////////////////
//  EditorPrefs

// EditorPrefs contains editor preferences.  It can also be set
// from ki.Props style properties.
type EditorPrefs struct { //gti:add

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
type FavPathItem struct { //gti:add

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

// Open file paths from a json-formatted file.
func (pf *FilePaths) Open(filename string) error { //gti:add
	return grr.Log(jsons.Open(pf, filename))
}

// Save file paths to a json-formatted file.
func (pf *FilePaths) Save(filename string) error { //gti:add
	return grr.Log(jsons.Save(pf, filename))
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
	pdir := GokiDataDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Save(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

// OpenPaths loads the active SavedPaths from prefs dir
func OpenPaths() {
	// remove to be sure we don't have duplicate extras
	StringsRemoveExtras((*[]string)(&SavedPaths), SavedPathsExtras)
	pdir := GokiDataDir()
	pnm := filepath.Join(pdir, SavedPathsFileName)
	SavedPaths.Open(pnm)
	// add back after save
	StringsAddExtras((*[]string)(&SavedPaths), SavedPathsExtras)
}

//////////////////////////////////////////////////////////////////
//  PrefsDetailed

// TODO: make all of the MSec things time.Duration

// PrefsDetailed are more detailed params not usually customized, but
// available for those who really care..
type PrefsDetailed struct { //gti:add

	// the maximum height of any menu popup panel in units of font height -- scroll bars are enforced beyond that size.
	MenuMaxHeight int `def:"30" min:"5" step:"1"`

	// the amount of time to wait before offering completions
	CompleteWaitDuration time.Duration `def:"0" min:"0" max:"10000" step:"10"`

	// the maximum number of completions offered in popup
	CompleteMaxItems int `def:"25" min:"5" step:"1"`

	// time interval for cursor blinking on and off -- set to 0 to disable blinking
	CursorBlinkTime time.Duration `def:"500" min:"0" max:"1000" step:"5"`

	// is amount of time to wait before trying to autoscroll again
	LayoutAutoScrollDelay time.Duration `def:"25" min:"1" step:"5"`

	// number of steps to take in PageUp / Down events in terms of number of items
	LayoutPageSteps int `def:"10" min:"1" step:"1"`

	// the amount of time between keypresses to combine characters into name to search for within layout -- starts over after this delay
	LayoutFocusNameTimeout time.Duration `def:"500" min:"0" max:"5000" step:"20"`

	// the amount of time since last focus name event to allow tab to focus on next element with same name.
	LayoutFocusNameTabTime time.Duration `def:"2000" min:"10" max:"10000" step:"100"`

	// open dialogs in separate windows -- else do as popups in main window
	DialogsSepRenderWin bool `def:"true"`

	// Maximum amount of clipboard history to retain
	TextEditorClipHistMax int `def:"100" min:"0" max:"1000" step:"5"`

	// maximum number of lines to look for matching scope syntax (parens, brackets)
	TextBufMaxScopeLines int `def:"100" min:"10" step:"10"`

	// text buffer max lines to use diff-based revert to more quickly update e.g., after file has been reformatted
	TextBufDiffRevertLines int `def:"10000" min:"0" step:"1000"`

	// text buffer max diffs to use diff-based revert to more quickly update e.g., after file has been reformatted -- if too many differences, just revert
	TextBufDiffRevertDiffs int `def:"20" min:"0" step:"1"`

	// number of milliseconds to wait before starting a new background markup process, after text changes within a single line (always does after line insertion / deletion)
	TextBufMarkupDelayMSec int `def:"1000" min:"100" step:"100"`

	// the number of map elements at or below which an inline representation of the map will be presented -- more convenient for small #'s of props
	MapInlineLen int `def:"2" min:"1" step:"1"`

	// the number of elemental struct fields at or below which an inline representation of the struct will be presented -- more convenient for small structs
	StructInlineLen int `def:"4" min:"2" step:"1"`

	// the number of slice elements below which inline will be used
	SliceInlineLen int `def:"4" min:"2" step:"1"`

	// flag that is set by StructView by virtue of changeflag tag, whenever an edit is made.  Used to drive save menus etc.
	Changed bool `view:"-" changeflag:"+" json:"-" toml:"-" xml:"-"`
}

// PrefsDet are the overall detailed preferences
var PrefsDet = PrefsDetailed{}

// PrefsDetailedFileName is the name of the detailed preferences file in GoGi prefs directory
var PrefsDetailedFileName = "prefs_det.toml"

// Open detailed preferences from GoGi standard prefs directory
func (pf *PrefsDetailed) Open() error { //gti:add
	pdir := GokiDataDir()
	pnm := filepath.Join(pdir, PrefsDetailedFileName)
	err := grr.Log(tomls.Open(pf, pnm))
	pf.Changed = false
	return err
}

// Save saves current preferences to standard prefs_det.toml file, which is auto-loaded at startup
func (pf *PrefsDetailed) Save() error { //gti:add
	pdir := GokiDataDir()
	pnm := filepath.Join(pdir, PrefsDetailedFileName)
	err := grr.Log(tomls.Save(pf, pnm))
	pf.Changed = false
	return err
}

// Defaults gets current values of parameters, which are effectively
// defaults
func (pf *PrefsDetailed) Defaults() {
	pf.MenuMaxHeight = MenuMaxHeight
	pf.CompleteWaitDuration = CompleteWaitDuration
	pf.CompleteMaxItems = CompleteMaxItems
	pf.CursorBlinkTime = CursorBlinkTime
	pf.LayoutAutoScrollDelay = LayoutAutoScrollDelay
	pf.LayoutPageSteps = LayoutPageSteps
	pf.LayoutFocusNameTimeout = LayoutFocusNameTimeout
	pf.LayoutFocusNameTabTime = LayoutFocusNameTabTime
	pf.MenuMaxHeight = MenuMaxHeight
	if TheViewIFace != nil {
		TheViewIFace.PrefsDetDefaults(pf)
	}
	// in giv:
	// TextEditorClipHistMax
	// TextBuf*
	// MapInlineLen
	// StructInlineLen
	// SliceInlineLen
}

// Apply detailed preferences to all the relevant settings.
func (pf *PrefsDetailed) Apply() { //gti:add
	MenuMaxHeight = pf.MenuMaxHeight
	CompleteWaitDuration = pf.CompleteWaitDuration
	CompleteMaxItems = pf.CompleteMaxItems
	CursorBlinkTime = pf.CursorBlinkTime
	LayoutFocusNameTimeout = pf.LayoutFocusNameTimeout
	LayoutFocusNameTabTime = pf.LayoutFocusNameTabTime
	MenuMaxHeight = pf.MenuMaxHeight
	if TheViewIFace != nil {
		TheViewIFace.PrefsDetApply(pf)
	}
	// in giv:
	// TextEditorClipHistMax = pf.TextEditorClipHistMax
	// TextBuf*
	// MapInlineLen
	// StructInlineLen
	// SliceInlineLen
}

//////////////////////////////////////////////////////////////////
//  PrefsDebug

// StrucdtViewIfDebug is a debug flag for getting error messages on
// viewif struct tag directives in the giv.StructView.
var StructViewIfDebug = false

// DebugSettingsData is the data type for debugging settings.
type DebugSettingsData struct { //gti:add
	SettingsBase

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
	Changed bool `view:"-" changeflag:"+" json:"-" toml:"-" xml:"-"`
}

// DebugSettings are the currently active debugging settings
var DebugSettings = &DebugSettingsData{
	SettingsBase: SettingsBase{
		File: filepath.Join("goki", "debug-settings.toml"),
	},
}

// Connect connects debug fields with actual variables controlling debugging
func (pf *DebugSettingsData) Connect() {
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

// Profile toggles profiling of program on or off, which does both
// targeted and global CPU and Memory profiling.
func (pf *DebugSettingsData) Profile() { //gti:add
	ProfileToggle()
}
