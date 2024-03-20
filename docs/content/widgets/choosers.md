# Choosers

Cogent Core provides highly customizable choosers that allow users to choose one option among a list of items.

You can set the items of a chooser from a list of strings:

```Go
gi.NewChooser(parent).SetStrings("macOS", "Windows", "Linux")
```

If you need to customize the items more, you can use a list of [[gi.ChooserItem]] objects:

```Go
gi.NewChooser(parent).SetItems(
    gi.ChooserItem{Value: "Computer", Icon: icons.Computer, Tooltip: "Use a computer"},
    gi.ChooserItem{Value: "Phone", Icon: icons.Smartphone, Tooltip: "Use a phone"},
)
```

You can set the placeholder value of a chooser:

```Go
gi.NewChooser(parent).SetPlaceholder("Choose a platform").SetStrings("macOS", "Windows", "Linux")
```

You can set the starting value of a chooser:

```Go
gi.NewChooser(parent).SetStrings("Apple", "Orange", "Strawberry").SetCurrentValue("Orange")
```

You can make a chooser outlined instead of filled:

```Go
gi.NewChooser(parent).SetType(gi.ChooserOutlined).SetStrings("Apple", "Orange", "Strawberry")
```

You can add an icon to a chooser:

```Go
gi.NewChooser(parent).SetIcon(icons.Sort).SetStrings("Newest", "Oldest", "Trending")
```