+++
Categories = ["Resources"]
+++

**Settings** are the preferences of the user that apply to all Cogent Core apps, such as the color theme, the zoom, and the keyboard shortcuts, along with any settings of a particular app. The user can edit them in the settings window, which is opened from the **Settings** item of the app menu, or with the `Command+,` keyboard shortcut (`Control+,` on non-macOS platforms; see [[keymaps]]). You can also open it with [[doc:core.SettingsWindow]]:

```Go
core.NewFuncButton(b).SetFunc(core.SettingsWindow)
```

Changes in the settings window apply right away, and are saved automatically.

## Standard settings

The standard settings are in [[doc:core.AllSettings]], each of which is shown as a tab of the settings window:

* **Appearance** ([[doc:core.AppearanceSettings]]): the color theme (light, dark, or automatic) and primary color, the zoom (overall, and for each screen), the spacing, and the font sizes, along with the highlighting style of code and the fonts of text.

* **System** ([[doc:core.SystemSettings]]): the keyboard shortcut map ([[keymaps]]), the settings of text editors, the clock, information about the user, and the favorite paths of the [[file picker]].

* **Printer** ([[doc:text/printer.Settings]]): the page size, margins, fonts, and line height used for printing and generating PDFs, as for [[content]] pages.

* **Timing** ([[doc:core.TimingSettings]]): durations and distances of user interactions, such as the double click interval, the scroll wheel speed, and the times for long presses and hovers.

* **Debug** ([[doc:core.DebugSettings]]): options for printing traces of updates, rendering, [[layout]], and [[events]], for debugging.

Settings that affect the look of the whole app include:

* **Zoom** scales everything, and can be changed at any time with the `Control+` and `Control-` keyboard shortcuts. A zoom can be saved for each screen, so that screens of different sizes and resolutions can use different zoom levels.

* **Spacing** scales the padding, margins, and gaps of all [[widget]]s (see [[layout#spacing]]). Lower values lead to a denser layout. The default is 60% on desktop platforms and 100% on mobile platforms.

* **Font size** scales the size of all text, and **docs font size** scales the text of documentation contexts such as [[content]] pages, relative to that.

## Where settings are saved

Each settings value is saved in its own file in [TOML](https://toml.io) format (unless the file name ends in `.json`), in the Cogent Core data directory, [[doc:system.App.CogentCoreDataDir]]. That is `Cogent Core` in the data directory of the platform:

* macOS: `~/Library/Cogent Core`
* Linux: `~/.config/Cogent Core`
* Windows: `AppData/Roaming/Cogent Core` in the home directory of the user

For example, the appearance settings are in `appearance-settings.toml`. Only settings that differ from their default values are saved, so a file may be nearly empty.

## App settings

Apps can add their own settings, which are shown as another tab in the settings window, and saved and loaded like the standard ones. Settings are a type that implements the [[doc:core.Settings]] interface, which is easiest to do by embedding [[doc:core.SettingsBase]], which has the name and file of the settings. The settings are added with [[doc:core.AddAppSettings]], typically in an `init` function, before the settings are loaded when the app starts:

```go
func init() {
	core.AddAppSettings(Settings)
}

// Settings are the settings of the app.
var Settings = &SettingsData{
	SettingsBase: core.SettingsBase{
		Name: "My app",
		File: filepath.Join(core.TheApp.DataDir(), "My app", "settings.toml"),
	},
}

// SettingsData is the data type of the settings of the app.
type SettingsData struct {
	core.SettingsBase

	// ShowGrid is whether to show the grid.
	ShowGrid bool `default:"true"`

	// GridSize is the size of the grid.
	GridSize int `default:"10" min:"1"`
}
```

The fields are shown in a [[form]], so the usual [[struct tags]] apply, including `default` for the default values, which are set each time the settings are loaded, before the saved values are read. The user can reset all of the settings to their defaults with **Reset settings** in the menu of the toolbar of the settings window.

The [[doc:core.Settings]] interface also has these methods, which the settings type can implement:

* `Defaults` sets the default values, for those that are not simple `default` struct tags.

* `Apply` applies the settings to the app after they are loaded or changed, such as by updating the widgets that depend on them.

* `MakeToolbar` adds settings-related actions to the toolbar of the settings tab.

To save the settings after changing them from code, use [[doc:core.SaveSettings]]. [[doc:core.UpdateSettings]] applies, saves, and updates the app after a change.
