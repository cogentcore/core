# Toolbars

Cogent Core provides an extensible system of powerful toolbars that allows you to create responsive toolbars that work on all platforms.

All toolbars use the [[tree.Plan]] system through [[core.WidgetBase.Maker]]. This ensures that toolbars can always be dynamic and responsive.

You can make a standalone toolbar and add elements to it:

```Go
tb := core.NewToolbar(b)
tb.Maker(func(p *tree.Plan) {
    tree.Add(p, func(w *core.Button) {
        w.SetText("Build")
    })
    tree.Add(p, func(w *core.Button) {
        w.SetText("Run")
    })
})
```

You can add any types of widgets to toolbars, although [buttons](../basic/buttons) and [func buttons](../other/func-buttons) are the most common:

```Go
tb := core.NewToolbar(b)
tb.Maker(func(p *tree.Plan) {
    tree.Add(p, func(w *core.FuncButton) {
        w.SetFunc(core.AppearanceSettings.SaveScreenZoom)
    })
    tree.Add(p, func(w *core.Switch) {
        w.SetText("Active")
    })
})
```

When you add more items to a toolbar than can fit on the screen, it places them in an overflow menu:

```Go
tb := core.NewToolbar(b)
tb.Maker(func(p *tree.Plan) {
    for i := range 30 {
        tree.AddAt(p, strconv.Itoa(i), func(w *core.Button) {
            w.SetText("Button "+strconv.Itoa(i))
        })
    }
})
```

You can also directly add items to the overflow menu of a toolbar:

```Go
tb := core.NewToolbar(b)
tb.Maker(func(p *tree.Plan) {
    tree.Add(p, func(w *core.Button) {
        w.SetText("Build")
    })
})
tb.AddOverflowMenu(func(m *core.Scene) {
    core.NewButton(m).SetText("Run")
})
```

Typically, you add elements to the main top app bar (see the toolbar at the top of this documentation for example) instead of making a standalone toolbar (in this example, `b` is the [[core.Body]]):

```go
b.AddAppBar(func(p *tree.Plan) {
    tree.Add(p, func(w *core.Button) {
        w.SetText("Build")
    })
    tree.Add(p, func(w *core.Button) {
        w.SetText("Run")
    })
})
```
