# Dialogs

Cogent Core provides various different types of customizable dialogs with support for any kind of content.

You can make a dialog with a text message:

```Go
bt := gi.NewButton(parent).SetText("Message")
bt.OnClick(func(e events.Event) {
    gi.MessageDialog(bt, "Something happened", "Message")
})
```

You can make a dialog with an error:

```Go
bt := gi.NewButton(parent).SetText("Error")
bt.OnClick(func(e events.Event) {
    gi.ErrorDialog(bt, errors.New("invalid encoding format"), "Error loading file")
})
```

If you need to make a dialog more complicated than a simple message or error dialog, you can use the [[gi.NewBody]] and [[gi.Body.NewDialog]] functions to construct a dialog with any content you want. For example, you can make a confirmation dialog:

```Go
bt := gi.NewButton(parent).SetText("Confirm")
bt.OnClick(func(e events.Event) {
    d := gi.NewBody().AddTitle("Confirm").AddText("Send message?")
    d.AddBottomBar(func(pw gi.Widget) {
        d.AddCancel(pw).OnClick(func(e events.Event) {
            gi.MessageSnackbar(bt, "Dialog canceled")
        })
        d.AddOk(pw).OnClick(func(e events.Event) {
            gi.MessageSnackbar(bt, "Dialog accepted")
        })
    })
    d.NewDialog(bt).Run()
})
```

## Close dialogs

Cogent Core supports dialogs that confirm that the user wants to close a scene when they try to close it, using the function [[gi.WidgetBase.AddCloseDialog]]. You can read the documentation of that function for more information on how it works, but a basic example is as follows: 

```go
b.AddCloseDialog(func(d *gi.Body) bool {
    d.AddTitle("Are you sure?").AddText("Are you sure you want to close the Cogent Core Demo?")
    d.AddBottomBar(func(pw gi.Widget) {
        d.AddOk(pw).SetText("Close").OnClick(func(e events.Event) {
            b.Scene.Close()
        })
    })
    return true
})
```