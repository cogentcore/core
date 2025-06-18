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
