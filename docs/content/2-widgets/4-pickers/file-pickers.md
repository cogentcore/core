# File pickers

Cogent Core provides powerful file pickers for selecting files that support sorting, navigation, searching, favorites, and history.

You can make a file picker:

```Go
core.NewFilePicker(b)
```

You can set the starting file of a file picker:

```Go
core.NewFilePicker(b).SetFilename(core.TheApp.DataDir())
```

You can detect when a user selects a file:

```Go
fp := core.NewFilePicker(b)
fp.OnSelect(func(e events.Event) {
    core.MessageSnackbar(fp, fp.SelectedFile())
})
```

You can make a button that opens a file picker dialog:

```Go
core.NewFileButton(b)
```

You can detect when the user selects a file using the dialog:

```Go
fb := core.NewFileButton(b)
fb.OnChange(func(e events.Event) {
    core.MessageSnackbar(fb, fb.Filename)
})
```
