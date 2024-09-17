Cogent Core provides context menus that allow users to take actions relevant to a widget. Users can open context menus by right clicking on a widget or pressing down on a widget for 500 milliseconds, so context menus work on all platforms.

You can add buttons to the context menu of a widget:

```Go
tf := core.NewTextField(b)
tf.AddContextMenu(func(m *core.Scene) {
    core.NewButton(m).SetText("Build")
    core.NewButton(m).SetText("Run")
})
```

You can remove all of the context menu buttons of a widget:

```Go
tf := core.NewTextField(b)
tf.ContextMenus = nil
```

Note that there is still a context menu in the example above since all widgets inherit the [[core.Scene]] context menu items, which consist of various important actions by default. You can remove these items if you want:

```go
myWidget.Scene.ContextMenus = nil
```
