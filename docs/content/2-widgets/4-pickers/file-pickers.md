# File pickers

Cogent Core provides powerful file pickers for selecting files that support sorting, navigation, searching, favorites, and history.

You can make a file picker:

```Go
core.NewFilePicker(parent)
```

You can detect when a user selects a file:

```Go
fp := core.NewFilePicker(parent)
fp.OnSelect(func(e events.Event) {
    core.MessageSnackbar(fp, fp.SelectedFile())
})
```

You can make a button that opens a file picker dialog:

```Go
core.NewFileButton(parent)
```

You can detect when the user selects a file using the dialog:

```Go
fb := core.NewFileButton(parent)
fb.OnChange(func(e events.Event) {
    core.MessageSnackbar(fb, fb.Filename)
})
```
