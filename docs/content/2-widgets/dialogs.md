# Dialogs

Cogent Core provides various different types of customizable dialogs with support for any kind of content.

You can make a dialog with a text message:

```Go
bt := core.NewButton(parent).SetText("Message")
bt.OnClick(func(e events.Event) {
    core.MessageDialog(bt, "Something happened", "Message")
})
```

You can make a dialog with an error:

```Go
bt := core.NewButton(parent).SetText("Error")
bt.OnClick(func(e events.Event) {
    core.ErrorDialog(bt, errors.New("invalid encoding format"), "Error loading file")
})
```

You can also construct a dialog with any content you want. For example, you can make a confirmation dialog:

```Go
bt := core.NewButton(parent).SetText("Confirm")
bt.OnClick(func(e events.Event) {
    d := core.NewBody().AddTitle("Confirm").AddText("Send message?")
    d.AddBottomBar(func(parent core.Widget) {
        d.AddCancel(parent).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Dialog canceled")
        })
        d.AddOK(parent).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Dialog accepted")
        })
    })
    d.RunDialog(bt)
})
```

You can make an input dialog:

```Go
bt := core.NewButton(parent).SetText("Input")
bt.OnClick(func(e events.Event) {
    d := core.NewBody().AddTitle("Input").AddText("What is your name?")
    tf := core.NewTextField(d)
    d.AddBottomBar(func(parent core.Widget) {
        d.AddCancel(parent)
        d.AddOK(parent).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Your name is "+tf.Text())
        })
    })
    d.RunDialog(bt)
})
```

You can make a dialog that takes up the entire window:

```Go
bt := core.NewButton(parent).SetText("Full window")
bt.OnClick(func(e events.Event) {
    d := core.NewBody().AddTitle("Full window dialog")
    d.RunFullDialog(bt)
})
```

You can make a dialog that opens in a new window on multi-window platforms (not mobile and web):

```Go
bt := core.NewButton(parent).SetText("New window")
bt.OnClick(func(e events.Event) {
    d := core.NewBody().AddTitle("New window dialog")
    d.RunDialogWindow(bt)
})
```

You can confirm that the user wants to close a scene when they try to close it:

```go
b.AddCloseDialog(func(d *core.Body) bool {
    d.AddTitle("Are you sure?").AddText("Are you sure you want to close the Cogent Core Demo?")
    d.AddBottomBar(func(parent core.Widget) {
        d.AddOK(parent).SetText("Close").OnClick(func(e events.Event) {
            b.Scene.Close()
        })
    })
    return true
})
```
