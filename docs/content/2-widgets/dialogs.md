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

You can also construct a dialog with any content you want. For example, you can make a confirmation dialog:

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

You can make an input dialog:

```Go
bt := gi.NewButton(parent).SetText("Input")
bt.OnClick(func(e events.Event) {
    d := gi.NewBody().AddTitle("Input").AddText("What is your name?")
    tf := gi.NewTextField(d)
    d.AddBottomBar(func(pw gi.Widget) {
        d.AddCancel(pw)
        d.AddOk(pw).OnClick(func(e events.Event) {
            gi.MessageSnackbar(bt, "Your name is "+tf.Text())
        })
    })
    d.NewDialog(bt).Run()
})
```

You can make a dialog that takes up the entire window:

```Go
bt := gi.NewButton(parent).SetText("Full window")
bt.OnClick(func(e events.Event) {
    d := gi.NewBody().AddTitle("Full window dialog")
    d.NewFullDialog(bt).Run()
})
```

You can make a dialog that opens in a new window on multi-window platforms (not mobile and web):

```Go
bt := gi.NewButton(parent).SetText("New window")
bt.OnClick(func(e events.Event) {
    d := gi.NewBody().AddTitle("New window dialog")
    d.NewDialog(bt).SetNewWindow(true).Run()
})
```

You can confirm that the user wants to close a scene when they try to close it:

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
