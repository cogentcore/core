# Buttons

Cogent Core provides interactive buttons that support text, icons, indicators, shortcuts, and menus. The standard behavior is to register a click event handler with [[gi.WidgetBase.OnClick]]. For example:

```Go
gi.NewButton(parent).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    gi.MessageSnackbar(parent, "Message sent")
})
```
