# Snackbars

Cogent Core provides customizable snackbars for displaying temporary information to the user.

You can make a snackbar with a text message:

```Go
bt := gi.NewButton(parent).SetText("Message")
bt.OnClick(func(e events.Event) {
    gi.MessageSnackbar(bt, "New messages loaded")
})
```

You can make a snackbar with an error:

```Go
bt := gi.NewButton(parent).SetText("Error")
bt.OnClick(func(e events.Event) {
    gi.ErrorSnackbar(bt, errors.New("file not found"), "Error loading page")
})
```

You can make a custom snackbar with a button and an icon:

```Go
bt := gi.NewButton(parent).SetText("Custom")
bt.OnClick(func(e events.Event) {
    gi.NewBody().AddSnackbarText("Files updated").
        AddSnackbarButton("Refresh", func(e events.Event) {
            gi.MessageSnackbar(bt, "Refreshed files")
        }).AddSnackbarIcon(icons.Close).NewSnackbar(bt).Run()
})
```
