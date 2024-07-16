Cogent Core provides highly customizable choosers that allow users to choose one option among a list of items.

You can set the items of a chooser from a list of strings:

```Go
core.NewChooser(b).SetStrings("macOS", "Windows", "Linux")
```

If you need to customize the items more, you can use a list of [[core.ChooserItem]] objects:

```Go
core.NewChooser(b).SetItems(
    core.ChooserItem{Value: "Computer", Icon: icons.Computer, Tooltip: "Use a computer"},
    core.ChooserItem{Value: "Phone", Icon: icons.Smartphone, Tooltip: "Use a phone"},
)
```

You can set the placeholder of a chooser:

```Go
core.NewChooser(b).SetPlaceholder("Choose a platform").SetStrings("macOS", "Windows", "Linux")
```

You can set the current value of a chooser:

```Go
core.NewChooser(b).SetStrings("Apple", "Orange", "Strawberry").SetCurrentValue("Orange")
```

You can make a chooser outlined instead of filled:

```Go
core.NewChooser(b).SetType(core.ChooserOutlined).SetStrings("Apple", "Orange", "Strawberry")
```

You can add an icon to a chooser:

```Go
core.NewChooser(b).SetIcon(icons.Sort).SetStrings("Newest", "Oldest", "Popular")
```

You can make a chooser a text field with automatic completion and validation support:

```Go
core.NewChooser(b).SetEditable(true).SetStrings("Newest", "Oldest", "Popular")
```

You can allow the user to add new items to a chooser:

```Go
core.NewChooser(b).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
```

You can make a chooser a text field and allow the user to add new items to it:

```Go
core.NewChooser(b).SetEditable(true).SetAllowNew(true).SetStrings("Newest", "Oldest", "Popular")
```

You can detect when the user changes the value of the chooser:

```Go
ch := core.NewChooser(b).SetStrings("Newest", "Oldest", "Popular")
ch.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Sorting by %v", ch.CurrentItem.Value))
})
```
