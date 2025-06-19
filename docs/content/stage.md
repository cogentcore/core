+++
Categories = ["Concepts"]
+++

A **stage** is a window, dialog, or popup in which a [[scene]] takes place. It manages the options for that, such as sizing and positioning.

A stage is *not* a widget. The scene is the root widget, and the stage is a non-widget object that manages the scene. Metaphorically, a scene "takes place" on a stage.

In simple end-user code, you never interact with a stage, since the [[body]] provides helper methods that handle everything:

```Go
b := core.NewBody("My App")
core.NewButton(b).SetText("Click me")
b.RunMainWindow()
```

However, under the hood, [[doc:core.Body.RunMainWindow]] is making a stage. Here is the expanded code:

```Go
b := core.NewBody("My App")
core.NewButton(b).SetText("Click me")
b.NewWindow().RunMain()
```

That call to [[doc:core.Body.NewWindow]] returns a [[doc:core.Stage]], which you can customize as documented below.

## Options

There are various options that you can set on a stage. For example, this code makes a fullscreen window on the second monitor:

```go
b.NewWindow().SetScreen(1).SetFullscreen(true).RunMain()
```

That code is for the main window. For a secondary window, you would replace `RunMain()` with `Run()`.

## Methods

You can also update the geometry of a window that is already running. For example, this code moves an already running window to the second monitor with position (30, 100) and size (1000, 1000):

```go
myWidget.Scene.SetGeometry(false, image.Pt(30, 100), image.Pt(1000, 1000), 1)
```
