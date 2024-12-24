+++
Categories = ["Widgets"]
+++

A **dialog** is a type of [[stage]] that displays content on top of the surrounding window.

## Simple

You can make a dialog with a [[text]] message:

```Go
bt := core.NewButton(b).SetText("Message")
bt.OnClick(func(e events.Event) {
    core.MessageDialog(bt, "Something happened", "Message")
})
```

You can make a dialog with an error:

```Go
bt := core.NewButton(b).SetText("Error")
bt.OnClick(func(e events.Event) {
    core.ErrorDialog(bt, errors.New("invalid encoding format"), "Error loading file")
})
```

## Custom

You can also construct a dialog with any content you want. For example, you can make a confirmation dialog:

```Go
bt := core.NewButton(b).SetText("Confirm")
bt.OnClick(func(e events.Event) {
    d := core.NewBody("Confirm")
    core.NewText(d).SetType(core.TextSupporting).SetText("Send message?")
    d.AddBottomBar(func(bar *core.Frame) {
        d.AddCancel(bar).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Dialog canceled")
        })
        d.AddOK(bar).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Dialog accepted")
        })
    })
    d.RunDialog(bt)
})
```

You can make an input dialog:

```Go
bt := core.NewButton(b).SetText("Input")
bt.OnClick(func(e events.Event) {
    d := core.NewBody("Input")
    core.NewText(d).SetType(core.TextSupporting).SetText("What is your name?")
    tf := core.NewTextField(d)
    d.AddBottomBar(func(bar *core.Frame) {
        d.AddCancel(bar)
        d.AddOK(bar).OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Your name is "+tf.Text())
        })
    })
    d.RunDialog(bt)
})
```

## Options

You can make a dialog that takes up the entire window:

```Go
bt := core.NewButton(b).SetText("Full window")
bt.OnClick(func(e events.Event) {
    d := core.NewBody("Full window dialog")
    d.RunFullDialog(bt)
})
```

You can make a dialog that opens in a new window on multi-window platforms (not mobile and web):

```Go
bt := core.NewButton(b).SetText("New window")
bt.OnClick(func(e events.Event) {
    d := core.NewBody("New window dialog")
    d.RunWindowDialog(bt)
})
```

## Close dialog

You can confirm that the user wants to close a scene when they try to close it:

```go
b.AddCloseDialog(func(d *core.Body) bool {
    d.SetTitle("Are you sure?")
    core.NewText(d).SetType(core.TextSupporting).SetText("Are you sure you want to close the Cogent Core Demo?")
    d.AddBottomBar(func(bar *core.Frame) {
        d.AddOK(bar).SetText("Close").OnClick(func(e events.Event) {
            b.Scene.Close()
        })
    })
    return true
})
```
