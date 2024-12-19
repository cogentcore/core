**Events** contains a list of important [[event]] types. You can also see the API documentation for an [exhaustive list](https://pkg.go.dev/cogentcore.org/core/events#Types) of event types.

## Mouse

Mouse events are triggered by the mouse/trackpad/touchpad on desktop platforms, and by finger motions on mobile platforms.

### Click

A click event is triggered when a user presses down and then up in a short period of time on a [[doc:styles/abilities.Clickable]] widget. Click events are often handled on [[button]]s:

```Go
core.NewButton(b).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Message sent")
})
```
