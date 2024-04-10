# Choosers

Cogent Core provides highly customizable choosers that allow users to choose one option among a list of items.

You can set the items of a chooser from a list of strings:

```Go
core.NewChooser(parent).SetStrings("macOS", "Windows", "Linux")
```

If you need to customize the items more, you can use a list of [[core.ChooserItem]] objects:

```Go
core.NewChooser(parent).SetItems(
    core.ChooserItem{Value: "Computer", Icon: icons.Computer, Tooltip: "Use a computer"},
    core.ChooserItem{Value: "Phone", Icon: icons.Smartphone, Tooltip: "Use a phone"},
)
```

You can set the placeholder of a chooser:

```Go
core.NewChooser(parent).SetPlaceholder("Choose a platform").SetStrings("macOS", "Windows", "Linux")
```

You can set the starting value of a chooser:

```Go
core.NewChooser(parent).SetStrings("Apple", "Orange", "Strawberry").SetCurrentValue("Orange")
```

You can make a chooser outlined instead of filled:

```Go
core.NewChooser(parent).SetType(core.ChooserOutlined).SetStrings("Apple", "Orange", "Strawberry")
```

You can add an icon to a chooser:

```Go
core.NewChooser(parent).SetIcon(icons.Sort).SetStrings("Newest", "Oldest", "Popular")
```

You can make a chooser a text field with automatic completion and validation support:

```Go
core.NewChooser(parent).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
```

You can allow the user to add new items to a chooser:

```Go
core.NewChooser(parent).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
```

You can make a chooser a text field and allow the user to add new items to it:

```Go
core.NewChooser(parent).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
```

You can detect when the user changes the value of the chooser:

```Go
ch := core.NewChooser(parent).SetStrings("Newest", "Oldest", "Popular")
ch.OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("Sorting by %v", ch.CurrentItem.Value))
})
```