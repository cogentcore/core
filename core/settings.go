// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"
	"image/color"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"time"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/colors/matcolor"
	"cogentcore.org/core/events"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/system"
	"cogentcore.org/core/xgo/option"
	"cogentcore.org/core/xio/jsons"
	"cogentcore.org/core/xio/tomls"
)

// AllSettings is a global slice containing all of the user [Settings]
// that the user will see in the settings window. It contains the base Cogent Core
// settings by default and should be modified by other apps to add their
// app settings.
var AllSettings = []Settings{AppearanceSettings, SystemSettings, DeviceSettings, DebugSettings}

// Settings is the interface that describes the functionality common to all settings data types.
type Settings interface {

	// Label returns the label text for the settings.
	Label() string

	// Filename returns the full filename/filepath at which the settings are stored.
	Filename() string

	// Defaults sets the default values for all of the settings.
	Defaults()

	// Apply does anything necessary to apply the settings to the app.
	Apply()

	// ConfigToolbar is an optional method that settings objects can implement in order to
	// configure the settings view toolbar with settings-related actions that the user can
	// perform.
	ConfigToolbar(tb *Toolbar)
}

// SettingsOpener is an optional additional interface that
// [Settings] can satisfy to customize the behavior of [OpenSettings].
type SettingsOpener interface {
	Settings

	// Open opens the settings
	Open() error
}

// SettingsSaver is an optional additional interface that
// [Settings] can satisfy to customize the behavior of [SaveSettings].
type SettingsSaver interface {
	Settings

	// Save saves the settings
	Save() error
}

// SettingsBase contains base settings logic that other settings data types can extend.
type SettingsBase struct {

	// Name is the name of the settings.
	Name string `view:"-" save:"-"`

	// File is the full filename/filepath at which the settings are stored.
	File string `view:"-" save:"-"`
}

// Label returns the label text for the settings.
func (sb *SettingsBase) Label() string {
	return sb.Name
}

// Filename returns the full filename/filepath at which the settings are stored.
func (sb *SettingsBase) Filename() string {
	return sb.File
}

// Defaults does nothing by default and can be extended by other settings data types.
func (sb *SettingsBase) Defaults() {}

// Apply does nothing by default and can be extended by other settings data types.
func (sb *SettingsBase) Apply() {}

// ConfigToolbar does nothing by default and can be extended by other settings data types.
func (sb *SettingsBase) ConfigToolbar(tb *Toolbar) {}

// OpenSettings opens the given settings from their [Settings.Filename].
// The settings are assumed to be in TOML unless they have a .json file
// extension. If they satisfy the [SettingsOpener] interface,
// [SettingsOpener.Open] will be used instead.
func OpenSettings(se Settings) error {
	if so, ok := se.(SettingsOpener); ok {
		return so.Open()
	}
	fnm := se.Filename()
	if filepath.Ext(fnm) == ".json" {
		return jsons.Open(se, fnm)
	}
	return tomls.Open(se, fnm)
}

// SaveSettings saves the given settings to their [Settings.Filename].
// The settings will be encoded in TOML unless they have a .json file
// extension. If they satisfy the [SettingsSaver] interface,
// [SettingsSaver.Save] will be used instead. Any non default
// fields are not saved, following [laser.NonDefaultFields].
func SaveSettings(se Settings) error {
	if ss, ok := se.(SettingsSaver); ok {
		return ss.Save()
	}
	fnm := se.Filename()
	ndf := laser.NonDefaultFields(se)
	if filepath.Ext(fnm) == ".json" {
		return jsons.Save(ndf, fnm)
	}
	return tomls.Save(ndf, fnm)
}

// ResetSettings resets the given settings to their default values.
func ResetSettings(se Settings) error {
	err := os.RemoveAll(se.Filename())
	if err != nil {
		return err
	}
	npv := laser.NonPtrValue(reflect.ValueOf(se))
	// we only reset the non-default fields to avoid removing the base
	// information (name, filename, etc)
	ndf := laser.NonDefaultFields(se)
	for f := range ndf {
		rf := npv.FieldByName(f)
		rf.Set(reflect.Zero(rf.Type()))
	}
	return LoadSettings(se)
}

// ResetAllSettings resets all of the settings to their default values.
func ResetAllSettings() error { //gti:add
	for _, se := range AllSettings {
		err := ResetSettings(se)
		if err != nil {
			return err
		}
	}
	UpdateAll()
	return nil
}

// LoadSettings sets the defaults of, opens, and applies the given settings.
// If they are not already saved, it saves them. It process their `default:` struct
// tags in addition to calling their [Settings.Default] method.
func LoadSettings(se Settings) error {
	grr.Log(laser.SetFromDefaultTags(se))
	se.Defaults()
	err := OpenSettings(se)
	// we always apply the settings even if we can't open them
	// to apply at least the default values
	se.Apply()
	if errors.Is(err, fs.ErrNotExist) {
		return nil // it is okay for settings to not be saved
	}
	return err
}

// LoadAllSettings sets the defaults of, opens, and applies [AllSettings].
func LoadAllSettings() error {
	errs := []error{}
	for _, se := range AllSettings {
		err := LoadSettings(se)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// SaveAllSettings saves [AllSettings].
func SaveAllSettings() error {
	errs := []error{}
	for _, se := range AllSettings {
		err := SaveSettings(se)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// UpdateAll updates all windows and triggers a full render rebuild.
// It is typically called when user settings are changed.
func UpdateAll() { //gti:add
	gradient.Cache = nil // the cache is invalid now
	for _, w := range AllRenderWindows {
		rctx := w.MainStageMgr.RenderContext
		rctx.LogicalDPI = w.LogicalDPI()
		rctx.SetFlag(true, RenderRebuild) // trigger full rebuild
	}
}

// AppearanceSettings are the currently active global Cogent Core appearance settings.
var AppearanceSettings = &AppearanceSettingsData{
	SettingsBase: SettingsBase{
		Name: "Appearance",
		File: filepath.Join(TheApp.CogentCoreDataDir(), "appearance-settings.toml"),
	},
}

// AppearanceSettingsData is the data type for the global Cogent Core appearance settings.
type AppearanceSettingsData struct { //gti:add
	SettingsBase

	// the color theme
	Theme Themes `default:"Auto"`

	// the primary color used to generate the color scheme
	Color color.RGBA `default:"#4285f4"`

	// overall zoom factor as a percentage of the default zoom
	Zoom float32 `default:"100" min:"10" max:"500" step:"10" format:"%g%%"`

	// the overall spacing factor as a percentage of the default amount of spacing
	// (higher numbers lead to more space and lower numbers lead to higher density)
	Spacing float32 `default:"100" min:"10" max:"500" step:"10" format:"%g%%"`

	// the overall font size factor applied to all text as a percentage
	// of the default font size (higher numbers lead to larger text)
	FontSize float32 `default:"100" min:"10" max:"500" step:"10" format:"%g%%"`

	// the amount that alternating rows are highlighted when showing tabular data (set to 0 to disable zebra striping)
	ZebraStripes float32 `default:"0" min:"0" max:"100" step:"10" format:"%g%%"`

	// screen-specific settings, which will override overall defaults if set
	Screens map[string]ScreenSettings

	// text highlighting style / theme
	HiStyle HiStyleName `default:"emacs"`

	// default font family when otherwise not specified
	FontFamily FontName `default:"Roboto"`

	// default mono-spaced font family
	MonoFont FontName `default:"Roboto Mono"`
}

func (as *AppearanceSettingsData) ShouldShow(field string) bool {
	switch field {
	case "Color":
		return !ForceAppColor
	}
	return true
}

// AppColor is the default primary color used to generate the color
// scheme. The user can still change the primary color used to generate
// the color scheme through [AppearanceSettingsData.Color] unless
// [ForceAppColor] is set to true, but this value will always take
// effect if the settings color is the default value. It defaults to
// Google Blue (#4285f4).
var AppColor = color.RGBA{66, 133, 244, 255}

// ForceAppColor is whether to prevent the user from changing the color
// scheme and make it always based on [AppColor].
var ForceAppColor bool

func (as *AppearanceSettingsData) Apply() { //gti:add
	if ForceAppColor || (as.Color == color.RGBA{66, 133, 244, 255}) {
		colors.SetSchemes(AppColor)
	} else {
		colors.SetSchemes(as.Color)
	}
	switch as.Theme {
	case ThemeLight:
		colors.SetScheme(false)
	case ThemeDark:
		colors.SetScheme(true)
	case ThemeAuto:
		colors.SetScheme(system.TheApp.IsDark())
	}
	if as.HiStyle == "" {
		as.HiStyle = "emacs" // todo: need light / dark versions
	}

	// TODO(kai): move HiStyle to a separate text editor settings
	// if TheViewInterface != nil {
	// 	TheViewInterface.SetHiStyleDefault(as.HiStyle)
	// }

	as.ApplyDPI()
}

// ApplyDPI updates the screen LogicalDPI values according to current
// settings and zoom factor, and then updates all open windows as well.
func (as *AppearanceSettingsData) ApplyDPI() {
	// zoom is percentage, but LogicalDPIScale is multiplier
	system.LogicalDPIScale = as.Zoom / 100
	// fmt.Println("system ldpi:", system.LogicalDPIScale)
	n := system.TheApp.NScreens()
	for i := 0; i < n; i++ {
		sc := system.TheApp.Screen(i)
		if sc == nil {
			continue
		}
		if scp, ok := as.Screens[sc.Name]; ok {
			// zoom is percentage, but LogicalDPIScale is multiplier
			system.SetLogicalDPIScale(sc.Name, scp.Zoom/100)
		}
		sc.UpdateLogicalDPI()
	}
	for _, w := range AllRenderWindows {
		w.SystemWindow.SetLogicalDPI(w.SystemWindow.Screen().LogicalDPI)
		// this isn't DPI-related, but this is the most efficient place to do it
		w.SystemWindow.SetTitleBarIsDark(matcolor.SchemeIsDark)
	}
}

// DeleteSavedWindowGeoms deletes the file that saves the position and size of
// each window, by screen, and clear current in-memory cache. You shouldn't generally
// need to do this, but sometimes it is useful for testing or windows that are
// showing up in bad places that you can't recover from.
func (as *AppearanceSettingsData) DeleteSavedWindowGeoms() { //gti:add
	TheWindowGeometrySaver.DeleteAll()
}

// ZebraStripesWeight returns a 0 to 0.2 alpha opacity factor to use in computing
// a zebra stripe color.
func (as *AppearanceSettingsData) ZebraStripesWeight() float32 {
	return as.ZebraStripes * 0.002
}

// DeviceSettings are the global device settings.
var DeviceSettings = &DeviceSettingsData{
	SettingsBase: SettingsBase{
		Name: "Device",
		File: filepath.Join(TheApp.CogentCoreDataDir(), "device-settings.toml"),
	},
}

// SaveScreenZoom saves the current zoom factor for current screen.
func (as *AppearanceSettingsData) SaveScreenZoom() { //gti:add
	sc := system.TheApp.Screen(0)
	sp, ok := as.Screens[sc.Name]
	if !ok {
		sp = ScreenSettings{}
	}
	sp.Zoom = as.Zoom
	if as.Screens == nil {
		as.Screens = make(map[string]ScreenSettings)
	}
	as.Screens[sc.Name] = sp
	grr.Log(SaveSettings(as))
}

// DeviceSettingsData is the data type for the device settings.
type DeviceSettingsData struct { //gti:add
	SettingsBase

	// The keyboard shortcut map to use
	KeyMap keyfun.MapName

	// The keyboard shortcut maps available as options for Key map.
	// If you do not want to have custom key maps, you should leave
	// this unset so that you always have the latest standard key maps.
	KeyMaps option.Option[keyfun.Maps]

	// The maximum time interval between button press events to count as a double-click
	DoubleClickInterval time.Duration `default:"500ms" min:"100ms" step:"50ms"`

	// How fast the scroll wheel moves, which is typically pixels per wheel step
	// but units can be arbitrary. It is generally impossible to standardize speed
	// and variable across devices, and we don't have access to the system settings,
	// so unfortunately you have to set it here.
	ScrollWheelSpeed float32 `default:"1" min:"0.01" step:"1"`

	// The amount of time to wait before initiating a slide event
	// (as opposed to a basic press event)
	SlideStartTime time.Duration `default:"50ms" min:"5ms" max:"1s" step:"5ms"`

	// The amount of time to wait before initiating a drag (drag and drop) event
	// (as opposed to a basic press or slide event)
	DragStartTime time.Duration `default:"250ms" min:"5ms" max:"1s" step:"5ms"`

	// The amount of time to wait between each repeat click event,
	// when the mouse is pressed down.  The first click is 8x this.
	RepeatClickTime time.Duration `default:"100ms" min:"5ms" max:"1s" step:"5ms"`

	// The number of pixels that must be moved before initiating a slide/drag
	// event (as opposed to a basic press event)
	DragStartDistance int `default:"4" min:"0" max:"100" step:"1"`

	// The amount of time to wait before initiating a long hover event (e.g., for opening a tooltip)
	LongHoverTime time.Duration `default:"500ms" min:"10ms" max:"10s" step:"10ms"`

	// The maximum number of pixels that mouse can move and still register a long hover event
	LongHoverStopDistance int `default:"5" min:"0" max:"1000" step:"1"`

	// The amount of time to wait before initiating a long press event (e.g., for opening a tooltip)
	LongPressTime time.Duration `default:"500ms" min:"10ms" max:"10s" step:"10ms"`

	// The maximum number of pixels that mouse/finger can move and still register a long press event
	LongPressStopDistance int `default:"50" min:"0" max:"1000" step:"1"`
}

func (ds *DeviceSettingsData) Defaults() {
	ds.KeyMap = keyfun.DefaultMap
	ds.KeyMaps.Value = keyfun.AvailableMaps
}

func (ds *DeviceSettingsData) Apply() {
	if ds.KeyMaps.Valid {
		keyfun.AvailableMaps = ds.KeyMaps.Value
	}
	if ds.KeyMap != "" {
		keyfun.SetActiveMapName(ds.KeyMap)
	}

	events.ScrollWheelSpeed = ds.ScrollWheelSpeed
}

// ScreenSettings are the per-screen settings -- see [system.App.Screen] for
// info on the different screens -- these prefs are indexed by the Screen.Name
// -- settings here override those in the global settings.
type ScreenSettings struct { //gti:add

	// overall zoom factor as a percentage of the default zoom
	Zoom float32 `default:"100" min:"10" max:"1000" step:"10"`
}

// SystemSettings are the currently active Cogent Core system settings.
var SystemSettings = &SystemSettingsData{
	SettingsBase: SettingsBase{
		Name: "System",
		File: filepath.Join(TheApp.CogentCoreDataDir(), "system-settings.toml"),
	},
}

// SystemSettingsData is the data type of the global Cogent Core settings.
type SystemSettingsData struct { //gti:add
	SettingsBase

	// text editor settings
	Editor EditorSettings

	// whether to use a 24-hour clock (instead of AM and PM)
	Clock24 bool `label:"24-hour clock"`

	// SnackbarTimeout is the default amount of time until snackbars
	// disappear (snackbars show short updates about app processes
	// at the bottom of the screen)
	SnackbarTimeout time.Duration `default:"5s"`

	// only support closing the currently selected active tab; if this is set to true, pressing the close button on other tabs will take you to that tab, from which you can close it
	OnlyCloseActiveTab bool `default:"false"`

	// the limit of file size, above which user will be prompted before opening / copying, etc.
	BigFileSize int `default:"10000000"`

	// maximum number of saved paths to save in FileView
	SavedPathsMax int `default:"50"`

	// extra font paths, beyond system defaults -- searched first
	FontPaths []string

	// user info -- partially filled-out automatically if empty / when prefs first created
	User User

	// favorite paths, shown in FileViewer and also editable there
	FavPaths FavPaths

	// column to sort by in FileView, and :up or :down for direction -- updated automatically via FileView
	FileViewSort string `view:"-"`

	// the maximum height of any menu popup panel in units of font height;
	// scroll bars are enforced beyond that size.
	MenuMaxHeight int `default:"30" min:"5" step:"1"`

	// the amount of time to wait before offering completions
	CompleteWaitDuration time.Duration `default:"0ms" min:"0ms" max:"10s" step:"10ms"`

	// the maximum number of completions offered in popup
	CompleteMaxItems int `default:"25" min:"5" step:"1"`

	// time interval for cursor blinking on and off -- set to 0 to disable blinking
	CursorBlinkTime time.Duration `default:"500ms" min:"0ms" max:"1s" step:"5ms"`

	// The amount of time to wait before trying to autoscroll again
	LayoutAutoScrollDelay time.Duration `default:"25ms" min:"1ms" step:"5ms"`

	// number of steps to take in PageUp / Down events in terms of number of items
	LayoutPageSteps int `default:"10" min:"1" step:"1"`

	// the amount of time between keypresses to combine characters into name to search for within layout -- starts over after this delay
	LayoutFocusNameTimeout time.Duration `default:"500ms" min:"0ms" max:"5s" step:"20ms"`

	// the amount of time since last focus name event to allow tab to focus on next element with same name.
	LayoutFocusNameTabTime time.Duration `default:"2s" min:"10ms" max:"10s" step:"100ms"`

	// the number of map elements at or below which an inline representation
	// of the map will be presented, which is more convenient for small #'s of properties
	MapInlineLength int `default:"2" min:"1" step:"1"`

	// the number of elemental struct fields at or below which an inline representation
	// of the struct will be presented, which is more convenient for small structs
	StructInlineLength int `default:"4" min:"2" step:"1"`

	// the number of slice elements below which inline will be used
	SliceInlineLength int `default:"4" min:"2" step:"1"`
}

func (ss *SystemSettingsData) Defaults() {
	ss.FavPaths.SetToDefaults()
	ss.UpdateUser()
}

// Apply detailed settings to all the relevant settings.
func (ss *SystemSettingsData) Apply() { //gti:add
	if ss.FontPaths != nil {
		paths := append(ss.FontPaths, paint.FontPaths...)
		paint.FontLibrary.InitFontPaths(paths...)
	} else {
		paint.FontLibrary.InitFontPaths(paint.FontPaths...)
	}

	np := len(ss.FavPaths)
	for i := 0; i < np; i++ {
		if ss.FavPaths[i].Ic == "" {
			ss.FavPaths[i].Ic = "folder"
		}
	}
}

func (ss *SystemSettingsData) Open() error {
	fnm := ss.Filename()
	err := tomls.Open(ss, fnm)
	if len(ss.FavPaths) == 0 {
		ss.FavPaths.SetToDefaults()
	}
	return err
}

// TimeFormat returns the Go time format layout string that should
// be used for displaying times to the user, based on the value of
// [SystemSettingsData.Clock24].
func (ss *SystemSettingsData) TimeFormat() string {
	if ss.Clock24 {
		return "15:04"
	}
	return "3:04 PM"
}

// UpdateUser gets the user info from the OS
func (ss *SystemSettingsData) UpdateUser() {
	usr, err := user.Current()
	if err == nil {
		ss.User.User = *usr
	}
}

// User basic user information that might be needed for different apps
type User struct { //gti:add
	user.User

	// default email address -- e.g., for recording changes in a version control system
	Email string
}

// EditorSettings contains text editor settings.
type EditorSettings struct { //gti:add

	// size of a tab, in chars -- also determines indent level for space indent
	TabSize int `default:"4" xml:"tab-size"`

	// use spaces for indentation, otherwise tabs
	SpaceIndent bool `xml:"space-indent"`

	// wrap lines at word boundaries -- otherwise long lines scroll off the end
	WordWrap bool `default:"true" xml:"word-wrap"`

	// show line numbers
	LineNos bool `default:"true" xml:"line-nos"`

	// use the completion system to suggest options while typing
	Completion bool `default:"true" xml:"completion"`

	// suggest corrections for unknown words while typing
	SpellCorrect bool `default:"true" xml:"spell-correct"`

	// automatically indent lines when enter, tab, }, etc pressed
	AutoIndent bool `default:"true" xml:"auto-indent"`

	// use emacs-style undo, where after a non-undo command, all the current undo actions are added to the undo stack, such that a subsequent undo is actually a redo
	EmacsUndo bool `xml:"emacs-undo"`

	// colorize the background according to nesting depth
	DepthColor bool `default:"true" xml:"depth-color"`
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

	// the path of the favorite item
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
	{icons.Document, "Documents", "~/Documents"},
	{icons.Download, "Downloads", "~/Downloads"},
	{icons.Computer, "root", "/"},
}

//////////////////////////////////////////////////////////////////
//  FilePaths

// FilePaths represents a set of file paths.
type FilePaths []string

// RecentPaths are the recently opened paths in the file view.
var RecentPaths FilePaths

// Open file paths from a json-formatted file.
func (fp *FilePaths) Open(filename string) error { //gti:add
	return grr.Log(jsons.Open(fp, filename))
}

// Save file paths to a json-formatted file.
func (fp *FilePaths) Save(filename string) error { //gti:add
	return grr.Log(jsons.Save(fp, filename))
}

// AddPath inserts a path to the file paths (at the start), subject to max
// length -- if path is already on the list then it is moved to the start.
func (fp *FilePaths) AddPath(path string, max int) {
	StringsInsertFirstUnique((*[]string)(fp), path, max)
}

// SavedPathsFilename is the name of the saved file paths file in
// the Cogent Core data directory.
var SavedPathsFilename = "saved-paths.json"

// SaveRecentPaths saves the active RecentPaths to data dir
func SaveRecentPaths() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	grr.Log(RecentPaths.Save(pnm))
}

// OpenRecentPaths loads the active RecentPaths from data dir
func OpenRecentPaths() {
	pdir := TheApp.CogentCoreDataDir()
	pnm := filepath.Join(pdir, SavedPathsFilename)
	grr.Log(RecentPaths.Open(pnm))
}

//////////////////////////////////////////////////////////////////
//  DebugSettings

// DebugSettings are the currently active debugging settings
var DebugSettings = &DebugSettingsData{
	SettingsBase: SettingsBase{
		Name: "Debug",
		File: filepath.Join(TheApp.CogentCoreDataDir(), "debug-settings.toml"),
	},
}

// DebugSettingsData is the data type for debugging settings.
type DebugSettingsData struct { //gti:add
	SettingsBase

	// Print a trace of updates that trigger re-rendering
	UpdateTrace bool

	// Print a trace of the nodes rendering
	RenderTrace bool

	// Print a trace of all layouts
	LayoutTrace bool

	// Print more detailed info about the underlying layout computations
	LayoutTraceDetail bool

	// Print a trace of window events
	WinEventTrace bool

	// Print the stack trace leading up to win publish events
	// which are expensive
	WinRenderTrace bool

	// Print a trace of window geometry saving / loading functions
	WinGeomTrace bool

	// Print a trace of keyboard events
	KeyEventTrace bool

	// Print a trace of event handling
	EventTrace bool

	// Print a trace of focus changes
	FocusTrace bool

	// Print a trace of DND event handling
	DNDTrace bool

	// Print a trace of Go language completion and lookup process
	GoCompleteTrace bool

	// Print a trace of Go language type parsing and inference process
	GoTypeTrace bool
}

func (db *DebugSettingsData) Defaults() {
	// TODO(kai/binsize): figure out how to do this without dragging in pi langs dependency
	// db.GoCompleteTrace = golang.CompleteTrace
	// db.GoTypeTrace = golang.TraceTypes
}

func (db *DebugSettingsData) Apply() {
	// golang.CompleteTrace = db.GoCompleteTrace
	// golang.TraceTypes = db.GoTypeTrace
}
