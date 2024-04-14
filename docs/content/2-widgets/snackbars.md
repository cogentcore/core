# Snackbars

Cogent Core provides customizable snackbars for displaying temporary information to the user.

You can make a snackbar with a text message:

```Go
bt := core.NewButton(parent).SetText("Message")
bt.OnClick(func(e events.Event) {
    core.MessageSnackbar(bt, "New messages loaded")
})
```

You can make a snackbar with an error:

```Go
bt := core.NewButton(parent).SetText("Error")
bt.OnClick(func(e events.Event) {
    core.ErrorSnackbar(bt, errors.New("file not found"), "Error loading page")
})
```

You can make a custom snackbar with a button and an icon:

```Go
bt := core.NewButton(parent).SetText("Custom")
bt.OnClick(func(e events.Event) {
    core.NewBody().AddSnackbarText("Files updated").
        AddSnackbarButton("Refresh", func(e events.Event) {
            core.MessageSnackbar(bt, "Refreshed files")
        }).AddSnackbarIcon(icons.Close).NewSnackbar(bt).Run()
})
```
