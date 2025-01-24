+++
Categories = ["Widgets"]
+++

A **file picker** is a [[widget]] for selecting files. It supports sorting, navigation, searching, favorites, and history.

For a nested filesystem tree, use a [[file tree]] instead.

## Properties

You can make a file picker:

```Go
core.NewFilePicker(b)
```

You can set the starting file of a file picker:

```Go
core.NewFilePicker(b).SetFilename(core.TheApp.DataDir())
```

## Events

You can detect when a user selects a file:

```Go
fp := core.NewFilePicker(b)
fp.OnSelect(func(e events.Event) {
    core.MessageSnackbar(fp, fp.SelectedFile())
})
```

## File button

You can make a [[button]] that opens a file picker [[dialog]]:

```Go
core.NewFileButton(b)
```

You can detect when a user selects a file using the dialog:

```Go
fb := core.NewFileButton(b)
fb.OnChange(func(e events.Event) {
    core.MessageSnackbar(fb, fb.Filename)
})
```
