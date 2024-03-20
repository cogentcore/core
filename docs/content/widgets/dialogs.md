# Dialogs

Cogent Core provides various different types of customizable dialogs with support for any kind of content.

You can use the [[gi.MessageDialog]] helper function to make a dialog with a text message:

```Go
bt := gi.NewButton(parent).SetText("Message dialog")
bt.OnClick(func(e events.Event) {
    gi.MessageDialog(bt, "Something happened", "Message")
})
```

You can use the [[gi.ErrorDialog]] helper function to make a dialog with an error:

```Go
bt := gi.NewButton(parent).SetText("Error dialog")
bt.OnClick(func(e events.Event) {
    gi.ErrorDialog(bt, errors.New("invalid encoding format"), "Error loading file")
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