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
