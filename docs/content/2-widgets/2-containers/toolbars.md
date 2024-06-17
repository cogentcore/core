# Toolbars

Cogent Core provides an extensible system of powerful toolbars that allows you to create responsive toolbars that work on all platforms.

All toolbars use the [[core.Plan]] system through [[core.WidgetBase.Maker]]. This ensures that toolbars can always be dynamic and responsive.

You can make a standalone toolbar and add elements to it:

```Go
tb := core.NewToolbar(parent)
tb.Maker(func(p *core.Plan) {
    core.Add(p, func(w *core.Button) {
        w.SetText("Build")
    })
    core.Add(p, func(w *core.Button) {
        w.SetText("Run")
    })
})
```

You can add any types of widgets to toolbars, although [buttons](../basic/buttons) and [func buttons](../other/func-buttons) are the most common:

```Go
tb := core.NewToolbar(parent)
tb.Maker(func(p *core.Plan) {
    core.Add(p, func(w *core.FuncButton) {
        w.SetFunc(core.AppearanceSettings.SaveScreenZoom)
    })
    core.Add(p, func(w *core.Switch) {
        w.SetText("Active")
    })
})
```

When you add more items to a toolbar than can fit on the screen, it places them in an overflow menu:

```Go
tb := core.NewToolbar(parent)
tb.Maker(func(p *core.Plan) {
    for i := range 30 {
        core.AddAt(p, strconv.Itoa(i), func(w *core.Button) {
            w.SetText("Button "+strconv.Itoa(i))
        })
    }
})
```
