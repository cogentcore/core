+++
Categories = ["Widgets"]
+++

A **snackbar** displays temporary information to users.

## Simple

You can make a snackbar with a [[text]] message:

```Go
bt := core.NewButton(b).SetText("Message")
bt.OnClick(func(e events.Event) {
    core.MessageSnackbar(bt, "New messages loaded")
})
```

You can make a snackbar with an error:

```Go
bt := core.NewButton(b).SetText("Error")
bt.OnClick(func(e events.Event) {
    core.ErrorSnackbar(bt, errors.New("file not found"), "Error loading page")
})
```

## Custom

You can make a custom snackbar with a [[button]] and/or [[icon]]:

```Go
bt := core.NewButton(b).SetText("Custom")
bt.OnClick(func(e events.Event) {
    core.NewBody().AddSnackbarText("Files updated").
        AddSnackbarButton("Refresh", func(e events.Event) {
            core.MessageSnackbar(bt, "Refreshed files")
        }).AddSnackbarIcon(icons.Close).RunSnackbar(bt)
})
```
