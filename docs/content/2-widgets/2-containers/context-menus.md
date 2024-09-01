Cogent Core provides context menus that allow users to take actions relevant to a widget. Users can open context menus by right clicking on a widget or pressing down on a widget for 500 milliseconds, so context menus work on all platforms.

You can add buttons to the context menu of a widget:

```Go
tf := core.NewTextField(b)
tf.AddContextMenu(func(m *core.Scene) {
    core.NewButton(m).SetText("Build")
    core.NewButton(m).SetText("Run")
})
```
