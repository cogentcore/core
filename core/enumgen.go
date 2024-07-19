// Code generated by "core generate"; DO NOT EDIT.

package core

import (
	"cogentcore.org/core/enums"
)

var _ButtonTypesValues = []ButtonTypes{0, 1, 2, 3, 4, 5, 6}

// ButtonTypesN is the highest valid value for type ButtonTypes, plus one.
const ButtonTypesN ButtonTypes = 7

var _ButtonTypesValueMap = map[string]ButtonTypes{`Filled`: 0, `Tonal`: 1, `Elevated`: 2, `Outlined`: 3, `Text`: 4, `Action`: 5, `Menu`: 6}

var _ButtonTypesDescMap = map[ButtonTypes]string{0: `ButtonFilled is a filled button with a contrasting background color. It should be used for prominent actions, typically those that are the final in a sequence. It is equivalent to Material Design&#39;s filled button.`, 1: `ButtonTonal is a filled button, similar to [ButtonFilled]. It is used for the same purposes, but it has a lighter background color and less emphasis. It is equivalent to Material Design&#39;s filled tonal button.`, 2: `ButtonElevated is an elevated button with a light background color and a shadow. It is equivalent to Material Design&#39;s elevated button.`, 3: `ButtonOutlined is an outlined button that is used for secondary actions that are still important. It is equivalent to Material Design&#39;s outlined button.`, 4: `ButtonText is a low-importance button with no border, background color, or shadow when not being interacted with. It renders primary-colored text, and it renders a background color and shadow when hovered/focused/active. It should only be used for low emphasis actions, and you must ensure it stands out from the surrounding context sufficiently. It is equivalent to Material Design&#39;s text button, but it can also contain icons and other things.`, 5: `ButtonAction is a simple button that typically serves as a simple action among a series of other buttons (eg: in a toolbar), or as a part of another widget, like a spinner or snackbar. It has no border, background color, or shadow when not being interacted with. It inherits the text color of its parent, and it renders a background when hovered/focused/active. You must ensure it stands out from the surrounding context sufficiently. It is equivalent to Material Design&#39;s icon button, but it can also contain text and other things (and frequently does).`, 6: `ButtonMenu is similar to [ButtonAction], but it is designed for buttons located in popup menus.`}

var _ButtonTypesMap = map[ButtonTypes]string{0: `Filled`, 1: `Tonal`, 2: `Elevated`, 3: `Outlined`, 4: `Text`, 5: `Action`, 6: `Menu`}

// String returns the string representation of this ButtonTypes value.
func (i ButtonTypes) String() string { return enums.String(i, _ButtonTypesMap) }

// SetString sets the ButtonTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *ButtonTypes) SetString(s string) error {
	return enums.SetString(i, s, _ButtonTypesValueMap, "ButtonTypes")
}

// Int64 returns the ButtonTypes value as an int64.
func (i ButtonTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the ButtonTypes value from an int64.
func (i *ButtonTypes) SetInt64(in int64) { *i = ButtonTypes(in) }

// Desc returns the description of the ButtonTypes value.
func (i ButtonTypes) Desc() string { return enums.Desc(i, _ButtonTypesDescMap) }

// ButtonTypesValues returns all possible values for the type ButtonTypes.
func ButtonTypesValues() []ButtonTypes { return _ButtonTypesValues }

// Values returns all possible values for the type ButtonTypes.
func (i ButtonTypes) Values() []enums.Enum { return enums.Values(_ButtonTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i ButtonTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *ButtonTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "ButtonTypes")
}

var _ChooserTypesValues = []ChooserTypes{0, 1}

// ChooserTypesN is the highest valid value for type ChooserTypes, plus one.
const ChooserTypesN ChooserTypes = 2

var _ChooserTypesValueMap = map[string]ChooserTypes{`Filled`: 0, `Outlined`: 1}

var _ChooserTypesDescMap = map[ChooserTypes]string{0: `ChooserFilled represents a filled Chooser with a background color and a bottom border`, 1: `ChooserOutlined represents an outlined Chooser with a border on all sides and no background color`}

var _ChooserTypesMap = map[ChooserTypes]string{0: `Filled`, 1: `Outlined`}

// String returns the string representation of this ChooserTypes value.
func (i ChooserTypes) String() string { return enums.String(i, _ChooserTypesMap) }

// SetString sets the ChooserTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *ChooserTypes) SetString(s string) error {
	return enums.SetString(i, s, _ChooserTypesValueMap, "ChooserTypes")
}

// Int64 returns the ChooserTypes value as an int64.
func (i ChooserTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the ChooserTypes value from an int64.
func (i *ChooserTypes) SetInt64(in int64) { *i = ChooserTypes(in) }

// Desc returns the description of the ChooserTypes value.
func (i ChooserTypes) Desc() string { return enums.Desc(i, _ChooserTypesDescMap) }

// ChooserTypesValues returns all possible values for the type ChooserTypes.
func ChooserTypesValues() []ChooserTypes { return _ChooserTypesValues }

// Values returns all possible values for the type ChooserTypes.
func (i ChooserTypes) Values() []enums.Enum { return enums.Values(_ChooserTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i ChooserTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *ChooserTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "ChooserTypes")
}

var _LayoutPassesValues = []LayoutPasses{0, 1, 2}

// LayoutPassesN is the highest valid value for type LayoutPasses, plus one.
const LayoutPassesN LayoutPasses = 3

var _LayoutPassesValueMap = map[string]LayoutPasses{`SizeUpPass`: 0, `SizeDownPass`: 1, `SizeFinalPass`: 2}

var _LayoutPassesDescMap = map[LayoutPasses]string{0: ``, 1: ``, 2: ``}

var _LayoutPassesMap = map[LayoutPasses]string{0: `SizeUpPass`, 1: `SizeDownPass`, 2: `SizeFinalPass`}

// String returns the string representation of this LayoutPasses value.
func (i LayoutPasses) String() string { return enums.String(i, _LayoutPassesMap) }

// SetString sets the LayoutPasses value from its string representation,
// and returns an error if the string is invalid.
func (i *LayoutPasses) SetString(s string) error {
	return enums.SetString(i, s, _LayoutPassesValueMap, "LayoutPasses")
}

// Int64 returns the LayoutPasses value as an int64.
func (i LayoutPasses) Int64() int64 { return int64(i) }

// SetInt64 sets the LayoutPasses value from an int64.
func (i *LayoutPasses) SetInt64(in int64) { *i = LayoutPasses(in) }

// Desc returns the description of the LayoutPasses value.
func (i LayoutPasses) Desc() string { return enums.Desc(i, _LayoutPassesDescMap) }

// LayoutPassesValues returns all possible values for the type LayoutPasses.
func LayoutPassesValues() []LayoutPasses { return _LayoutPassesValues }

// Values returns all possible values for the type LayoutPasses.
func (i LayoutPasses) Values() []enums.Enum { return enums.Values(_LayoutPassesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i LayoutPasses) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *LayoutPasses) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "LayoutPasses")
}

var _MeterTypesValues = []MeterTypes{0, 1, 2}

// MeterTypesN is the highest valid value for type MeterTypes, plus one.
const MeterTypesN MeterTypes = 3

var _MeterTypesValueMap = map[string]MeterTypes{`Linear`: 0, `Circle`: 1, `Semicircle`: 2}

var _MeterTypesDescMap = map[MeterTypes]string{0: `MeterLinear indicates to render a meter that goes in a straight, linear direction, either horizontal or vertical, as specified by [styles.Style.Direction].`, 1: `MeterCircle indicates to render the meter as a circle.`, 2: `MeterSemicircle indicates to render the meter as a semicircle.`}

var _MeterTypesMap = map[MeterTypes]string{0: `Linear`, 1: `Circle`, 2: `Semicircle`}

// String returns the string representation of this MeterTypes value.
func (i MeterTypes) String() string { return enums.String(i, _MeterTypesMap) }

// SetString sets the MeterTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *MeterTypes) SetString(s string) error {
	return enums.SetString(i, s, _MeterTypesValueMap, "MeterTypes")
}

// Int64 returns the MeterTypes value as an int64.
func (i MeterTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the MeterTypes value from an int64.
func (i *MeterTypes) SetInt64(in int64) { *i = MeterTypes(in) }

// Desc returns the description of the MeterTypes value.
func (i MeterTypes) Desc() string { return enums.Desc(i, _MeterTypesDescMap) }

// MeterTypesValues returns all possible values for the type MeterTypes.
func MeterTypesValues() []MeterTypes { return _MeterTypesValues }

// Values returns all possible values for the type MeterTypes.
func (i MeterTypes) Values() []enums.Enum { return enums.Values(_MeterTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i MeterTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *MeterTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "MeterTypes")
}

var _ThemesValues = []Themes{0, 1, 2}

// ThemesN is the highest valid value for type Themes, plus one.
const ThemesN Themes = 3

var _ThemesValueMap = map[string]Themes{`Auto`: 0, `Light`: 1, `Dark`: 2}

var _ThemesDescMap = map[Themes]string{0: `ThemeAuto indicates to use the theme specified by the operating system`, 1: `ThemeLight indicates to use a light theme`, 2: `ThemeDark indicates to use a dark theme`}

var _ThemesMap = map[Themes]string{0: `Auto`, 1: `Light`, 2: `Dark`}

// String returns the string representation of this Themes value.
func (i Themes) String() string { return enums.String(i, _ThemesMap) }

// SetString sets the Themes value from its string representation,
// and returns an error if the string is invalid.
func (i *Themes) SetString(s string) error { return enums.SetString(i, s, _ThemesValueMap, "Themes") }

// Int64 returns the Themes value as an int64.
func (i Themes) Int64() int64 { return int64(i) }

// SetInt64 sets the Themes value from an int64.
func (i *Themes) SetInt64(in int64) { *i = Themes(in) }

// Desc returns the description of the Themes value.
func (i Themes) Desc() string { return enums.Desc(i, _ThemesDescMap) }

// ThemesValues returns all possible values for the type Themes.
func ThemesValues() []Themes { return _ThemesValues }

// Values returns all possible values for the type Themes.
func (i Themes) Values() []enums.Enum { return enums.Values(_ThemesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i Themes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *Themes) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "Themes") }

var _SizeClassesValues = []SizeClasses{0, 1, 2}

// SizeClassesN is the highest valid value for type SizeClasses, plus one.
const SizeClassesN SizeClasses = 3

var _SizeClassesValueMap = map[string]SizeClasses{`Compact`: 0, `Medium`: 1, `Expanded`: 2}

var _SizeClassesDescMap = map[SizeClasses]string{0: `SizeCompact is the size class for windows with a width less than 600dp, which typically happens on phones.`, 1: `SizeMedium is the size class for windows with a width between 600dp and 840dp inclusive, which typically happens on tablets.`, 2: `SizeExpanded is the size class for windows with a width greater than 840dp, which typically happens on desktop and laptop computers.`}

var _SizeClassesMap = map[SizeClasses]string{0: `Compact`, 1: `Medium`, 2: `Expanded`}

// String returns the string representation of this SizeClasses value.
func (i SizeClasses) String() string { return enums.String(i, _SizeClassesMap) }

// SetString sets the SizeClasses value from its string representation,
// and returns an error if the string is invalid.
func (i *SizeClasses) SetString(s string) error {
	return enums.SetString(i, s, _SizeClassesValueMap, "SizeClasses")
}

// Int64 returns the SizeClasses value as an int64.
func (i SizeClasses) Int64() int64 { return int64(i) }

// SetInt64 sets the SizeClasses value from an int64.
func (i *SizeClasses) SetInt64(in int64) { *i = SizeClasses(in) }

// Desc returns the description of the SizeClasses value.
func (i SizeClasses) Desc() string { return enums.Desc(i, _SizeClassesDescMap) }

// SizeClassesValues returns all possible values for the type SizeClasses.
func SizeClassesValues() []SizeClasses { return _SizeClassesValues }

// Values returns all possible values for the type SizeClasses.
func (i SizeClasses) Values() []enums.Enum { return enums.Values(_SizeClassesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i SizeClasses) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *SizeClasses) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "SizeClasses")
}

var _SliderTypesValues = []SliderTypes{0, 1}

// SliderTypesN is the highest valid value for type SliderTypes, plus one.
const SliderTypesN SliderTypes = 2

var _SliderTypesValueMap = map[string]SliderTypes{`Slider`: 0, `Scrollbar`: 1}

var _SliderTypesDescMap = map[SliderTypes]string{0: `SliderSlider indicates a standard, user-controllable slider for setting a numeric value.`, 1: `SliderScrollbar indicates a slider acting as a scrollbar for content. It has a [Slider.visiblePercent] factor that specifies the percent of the content currently visible, which determines the size of the thumb, and thus the range of motion remaining for the thumb Value ([Slider.visiblePercent] = 1 means thumb is full size, and no remaining range of motion). The content size (inside the margin and padding) determines the outer bounds of the rendered area.`}

var _SliderTypesMap = map[SliderTypes]string{0: `Slider`, 1: `Scrollbar`}

// String returns the string representation of this SliderTypes value.
func (i SliderTypes) String() string { return enums.String(i, _SliderTypesMap) }

// SetString sets the SliderTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *SliderTypes) SetString(s string) error {
	return enums.SetString(i, s, _SliderTypesValueMap, "SliderTypes")
}

// Int64 returns the SliderTypes value as an int64.
func (i SliderTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the SliderTypes value from an int64.
func (i *SliderTypes) SetInt64(in int64) { *i = SliderTypes(in) }

// Desc returns the description of the SliderTypes value.
func (i SliderTypes) Desc() string { return enums.Desc(i, _SliderTypesDescMap) }

// SliderTypesValues returns all possible values for the type SliderTypes.
func SliderTypesValues() []SliderTypes { return _SliderTypesValues }

// Values returns all possible values for the type SliderTypes.
func (i SliderTypes) Values() []enums.Enum { return enums.Values(_SliderTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i SliderTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *SliderTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "SliderTypes")
}

var _StageTypesValues = []StageTypes{0, 1, 2, 3, 4, 5}

// StageTypesN is the highest valid value for type StageTypes, plus one.
const StageTypesN StageTypes = 6

var _StageTypesValueMap = map[string]StageTypes{`WindowStage`: 0, `DialogStage`: 1, `MenuStage`: 2, `TooltipStage`: 3, `SnackbarStage`: 4, `CompleterStage`: 5}

var _StageTypesDescMap = map[StageTypes]string{0: `WindowStage is a MainStage that displays a [Scene] in a full window. One of these must be created first, as the primary app content, and it typically persists throughout. It fills the [renderWindow]. Additional windows can be created either within the same [renderWindow] on all platforms or in separate [renderWindow]s on desktop platforms.`, 1: `DialogStage is a MainStage that displays a [Scene] in a smaller dialog window on top of a [WindowStage], or in a full or separate window. It can be [Stage.Modal] or not.`, 2: `MenuStage is a PopupStage that displays a [Scene] typically containing [Button]s overlaid on a MainStage. It is typically [Stage.Modal] and [Stage.ClickOff], and closes when an button is clicked.`, 3: `TooltipStage is a PopupStage that displays a [Scene] with extra text info for a widget overlaid on a MainStage. It is typically [Stage.ClickOff] and not [Stage.Modal].`, 4: `SnackbarStage is a PopupStage that displays a [Scene] with text info and an optional additional button. It is displayed at the bottom of the screen. It is typically not [Stage.ClickOff] or [Stage.Modal], but has a [Stage.Timeout].`, 5: `CompleterStage is a PopupStage that displays a [Scene] with text completion options, spelling corrections, or other such dynamic info. It is typically [Stage.ClickOff], not [Stage.Modal], dynamically updating, and closes when something is selected or typing renders it no longer relevant.`}

var _StageTypesMap = map[StageTypes]string{0: `WindowStage`, 1: `DialogStage`, 2: `MenuStage`, 3: `TooltipStage`, 4: `SnackbarStage`, 5: `CompleterStage`}

// String returns the string representation of this StageTypes value.
func (i StageTypes) String() string { return enums.String(i, _StageTypesMap) }

// SetString sets the StageTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *StageTypes) SetString(s string) error {
	return enums.SetString(i, s, _StageTypesValueMap, "StageTypes")
}

// Int64 returns the StageTypes value as an int64.
func (i StageTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the StageTypes value from an int64.
func (i *StageTypes) SetInt64(in int64) { *i = StageTypes(in) }

// Desc returns the description of the StageTypes value.
func (i StageTypes) Desc() string { return enums.Desc(i, _StageTypesDescMap) }

// StageTypesValues returns all possible values for the type StageTypes.
func StageTypesValues() []StageTypes { return _StageTypesValues }

// Values returns all possible values for the type StageTypes.
func (i StageTypes) Values() []enums.Enum { return enums.Values(_StageTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i StageTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *StageTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "StageTypes")
}

var _SwitchTypesValues = []SwitchTypes{0, 1, 2, 3, 4}

// SwitchTypesN is the highest valid value for type SwitchTypes, plus one.
const SwitchTypesN SwitchTypes = 5

var _SwitchTypesValueMap = map[string]SwitchTypes{`switch`: 0, `chip`: 1, `checkbox`: 2, `radio-button`: 3, `segmented-button`: 4}

var _SwitchTypesDescMap = map[SwitchTypes]string{0: `SwitchSwitch indicates to display a switch as a switch (toggle slider).`, 1: `SwitchChip indicates to display a switch as chip (like Material Design&#39;s filter chip), which is typically only used in the context of [Switches].`, 2: `SwitchCheckbox indicates to display a switch as a checkbox.`, 3: `SwitchRadioButton indicates to display a switch as a radio button.`, 4: `SwitchSegmentedButton indicates to display a segmented button, which is typically only used in the context of [Switches].`}

var _SwitchTypesMap = map[SwitchTypes]string{0: `switch`, 1: `chip`, 2: `checkbox`, 3: `radio-button`, 4: `segmented-button`}

// String returns the string representation of this SwitchTypes value.
func (i SwitchTypes) String() string { return enums.String(i, _SwitchTypesMap) }

// SetString sets the SwitchTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *SwitchTypes) SetString(s string) error {
	return enums.SetString(i, s, _SwitchTypesValueMap, "SwitchTypes")
}

// Int64 returns the SwitchTypes value as an int64.
func (i SwitchTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the SwitchTypes value from an int64.
func (i *SwitchTypes) SetInt64(in int64) { *i = SwitchTypes(in) }

// Desc returns the description of the SwitchTypes value.
func (i SwitchTypes) Desc() string { return enums.Desc(i, _SwitchTypesDescMap) }

// SwitchTypesValues returns all possible values for the type SwitchTypes.
func SwitchTypesValues() []SwitchTypes { return _SwitchTypesValues }

// Values returns all possible values for the type SwitchTypes.
func (i SwitchTypes) Values() []enums.Enum { return enums.Values(_SwitchTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i SwitchTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *SwitchTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "SwitchTypes")
}

var _TabTypesValues = []TabTypes{0, 1, 2, 3, 4}

// TabTypesN is the highest valid value for type TabTypes, plus one.
const TabTypesN TabTypes = 5

var _TabTypesValueMap = map[string]TabTypes{`StandardTabs`: 0, `FunctionalTabs`: 1, `NavigationAuto`: 2, `NavigationBar`: 3, `NavigationDrawer`: 4}

var _TabTypesDescMap = map[TabTypes]string{0: `StandardTabs indicates to render the standard type of Material Design style tabs.`, 1: `FunctionalTabs indicates to render functional tabs like those in Google Chrome. These tabs take up less space and are the only kind that can be closed. They will also support being moved at some point.`, 2: `NavigationAuto indicates to render the tabs as either [NavigationBar] or [NavigationDrawer] if [WidgetBase.SizeClass] is [SizeCompact] or not, respectively. NavigationAuto should typically be used instead of one of the specific navigation types for better cross-platform compatability.`, 3: `NavigationBar indicates to render the tabs as a bottom navigation bar with text and icons.`, 4: `NavigationDrawer indicates to render the tabs as a side navigation drawer with text and icons.`}

var _TabTypesMap = map[TabTypes]string{0: `StandardTabs`, 1: `FunctionalTabs`, 2: `NavigationAuto`, 3: `NavigationBar`, 4: `NavigationDrawer`}

// String returns the string representation of this TabTypes value.
func (i TabTypes) String() string { return enums.String(i, _TabTypesMap) }

// SetString sets the TabTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *TabTypes) SetString(s string) error {
	return enums.SetString(i, s, _TabTypesValueMap, "TabTypes")
}

// Int64 returns the TabTypes value as an int64.
func (i TabTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the TabTypes value from an int64.
func (i *TabTypes) SetInt64(in int64) { *i = TabTypes(in) }

// Desc returns the description of the TabTypes value.
func (i TabTypes) Desc() string { return enums.Desc(i, _TabTypesDescMap) }

// TabTypesValues returns all possible values for the type TabTypes.
func TabTypesValues() []TabTypes { return _TabTypesValues }

// Values returns all possible values for the type TabTypes.
func (i TabTypes) Values() []enums.Enum { return enums.Values(_TabTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i TabTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *TabTypes) UnmarshalText(text []byte) error { return enums.UnmarshalText(i, text, "TabTypes") }

var _TextTypesValues = []TextTypes{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14}

// TextTypesN is the highest valid value for type TextTypes, plus one.
const TextTypesN TextTypes = 15

var _TextTypesValueMap = map[string]TextTypes{`DisplayLarge`: 0, `DisplayMedium`: 1, `DisplaySmall`: 2, `HeadlineLarge`: 3, `HeadlineMedium`: 4, `HeadlineSmall`: 5, `TitleLarge`: 6, `TitleMedium`: 7, `TitleSmall`: 8, `BodyLarge`: 9, `BodyMedium`: 10, `BodySmall`: 11, `LabelLarge`: 12, `LabelMedium`: 13, `LabelSmall`: 14}

var _TextTypesDescMap = map[TextTypes]string{0: `TextDisplayLarge is large, short, and important display text with a default font size of 57dp.`, 1: `TextDisplayMedium is medium-sized, short, and important display text with a default font size of 45dp.`, 2: `TextDisplaySmall is small, short, and important display text with a default font size of 36dp.`, 3: `TextHeadlineLarge is large, high-emphasis headline text with a default font size of 32dp.`, 4: `TextHeadlineMedium is medium-sized, high-emphasis headline text with a default font size of 28dp.`, 5: `TextHeadlineSmall is small, high-emphasis headline text with a default font size of 24dp.`, 6: `TextTitleLarge is large, medium-emphasis title text with a default font size of 22dp.`, 7: `TextTitleMedium is medium-sized, medium-emphasis title text with a default font size of 16dp.`, 8: `TextTitleSmall is small, medium-emphasis title text with a default font size of 14dp.`, 9: `TextBodyLarge is large body text used for longer passages of text with a default font size of 16dp.`, 10: `TextBodyMedium is medium-sized body text used for longer passages of text with a default font size of 14dp.`, 11: `TextBodySmall is small body text used for longer passages of text with a default font size of 12dp.`, 12: `TextLabelLarge is large text used for label text (like a caption or the text inside a button) with a default font size of 14dp.`, 13: `TextLabelMedium is medium-sized text used for label text (like a caption or the text inside a button) with a default font size of 12dp.`, 14: `TextLabelSmall is small text used for label text (like a caption or the text inside a button) with a default font size of 11dp.`}

var _TextTypesMap = map[TextTypes]string{0: `DisplayLarge`, 1: `DisplayMedium`, 2: `DisplaySmall`, 3: `HeadlineLarge`, 4: `HeadlineMedium`, 5: `HeadlineSmall`, 6: `TitleLarge`, 7: `TitleMedium`, 8: `TitleSmall`, 9: `BodyLarge`, 10: `BodyMedium`, 11: `BodySmall`, 12: `LabelLarge`, 13: `LabelMedium`, 14: `LabelSmall`}

// String returns the string representation of this TextTypes value.
func (i TextTypes) String() string { return enums.String(i, _TextTypesMap) }

// SetString sets the TextTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *TextTypes) SetString(s string) error {
	return enums.SetString(i, s, _TextTypesValueMap, "TextTypes")
}

// Int64 returns the TextTypes value as an int64.
func (i TextTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the TextTypes value from an int64.
func (i *TextTypes) SetInt64(in int64) { *i = TextTypes(in) }

// Desc returns the description of the TextTypes value.
func (i TextTypes) Desc() string { return enums.Desc(i, _TextTypesDescMap) }

// TextTypesValues returns all possible values for the type TextTypes.
func TextTypesValues() []TextTypes { return _TextTypesValues }

// Values returns all possible values for the type TextTypes.
func (i TextTypes) Values() []enums.Enum { return enums.Values(_TextTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i TextTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *TextTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "TextTypes")
}

var _TextFieldTypesValues = []TextFieldTypes{0, 1}

// TextFieldTypesN is the highest valid value for type TextFieldTypes, plus one.
const TextFieldTypesN TextFieldTypes = 2

var _TextFieldTypesValueMap = map[string]TextFieldTypes{`Filled`: 0, `Outlined`: 1}

var _TextFieldTypesDescMap = map[TextFieldTypes]string{0: `TextFieldFilled represents a filled [TextField] with a background color and a bottom border.`, 1: `TextFieldOutlined represents an outlined [TextField] with a border on all sides and no background color.`}

var _TextFieldTypesMap = map[TextFieldTypes]string{0: `Filled`, 1: `Outlined`}

// String returns the string representation of this TextFieldTypes value.
func (i TextFieldTypes) String() string { return enums.String(i, _TextFieldTypesMap) }

// SetString sets the TextFieldTypes value from its string representation,
// and returns an error if the string is invalid.
func (i *TextFieldTypes) SetString(s string) error {
	return enums.SetString(i, s, _TextFieldTypesValueMap, "TextFieldTypes")
}

// Int64 returns the TextFieldTypes value as an int64.
func (i TextFieldTypes) Int64() int64 { return int64(i) }

// SetInt64 sets the TextFieldTypes value from an int64.
func (i *TextFieldTypes) SetInt64(in int64) { *i = TextFieldTypes(in) }

// Desc returns the description of the TextFieldTypes value.
func (i TextFieldTypes) Desc() string { return enums.Desc(i, _TextFieldTypesDescMap) }

// TextFieldTypesValues returns all possible values for the type TextFieldTypes.
func TextFieldTypesValues() []TextFieldTypes { return _TextFieldTypesValues }

// Values returns all possible values for the type TextFieldTypes.
func (i TextFieldTypes) Values() []enums.Enum { return enums.Values(_TextFieldTypesValues) }

// MarshalText implements the [encoding.TextMarshaler] interface.
func (i TextFieldTypes) MarshalText() ([]byte, error) { return []byte(i.String()), nil }

// UnmarshalText implements the [encoding.TextUnmarshaler] interface.
func (i *TextFieldTypes) UnmarshalText(text []byte) error {
	return enums.UnmarshalText(i, text, "TextFieldTypes")
}
