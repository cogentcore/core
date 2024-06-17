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
