# Switches

Cogent Core provides switches for selecting one or more options out of a list of items, in addition to standalone switches for controlling a single bool value.

You can make a standalone switch with no label:

```Go
gi.NewSwitch(parent)
```

You can add a label to a standalone switch:

```Go
gi.NewSwitch(parent).SetText("Remember me")
```

You can make a standalone switch render as a checkbox:

```Go
gi.NewSwitch(parent).SetType(gi.SwitchCheckbox).SetText("Remember me")
```

You can make a standalone switch render as a radio button:

```Go
gi.NewSwitch(parent).SetType(gi.SwitchRadioButton).SetText("Remember me")
```

You can make a group of switches from a list of strings:

```Go
gi.NewSwitches(parent).SetStrings("Go", "Python", "C++")
```

If you need to customize the items more, you can use a list of [[gi.SwitchItem]] objects:

```Go
gi.NewSwitches(parent).SetItems(
    gi.SwitchItem{Label: "Go", Tooltip: "Elegant, fast, and easy-to-use"},
    gi.SwitchItem{Label: "Python", Tooltip: "Slow and duck-typed"},
    gi.SwitchItem{Label: "C++", Tooltip: "Hard to use and slow to compile"},
)
```

You can make switches mutually exclusive so that only one can be selected at a time:

```Go
gi.NewSwitches(parent).SetMutex(true).SetStrings("Go", "Python", "C++")
```

You can make switches render as chips:

```Go
gi.NewSwitches(parent).SetType(gi.SwitchChip).SetStrings("Go", "Python", "C++")
```

You can make switches render as checkboxes:

```Go
gi.NewSwitches(parent).SetType(gi.SwitchCheckbox).SetStrings("Go", "Python", "C++")
```

You can make switches render as radio buttons:

```Go
gi.NewSwitches(parent).SetType(gi.SwitchRadioButton).SetStrings("Go", "Python", "C++")
```

You can make switches render as segmented buttons:

```Go
gi.NewSwitches(parent).SetType(gi.SwitchSegmentedButton).SetStrings("Go", "Python", "C++")
```
