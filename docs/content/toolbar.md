+++
Categories = ["Widgets"]
+++

A **toolbar** is a [[frame]] that contains [[widget]]s for related actions and controls.

All toolbars use the [[plan]] system through [[doc:core.WidgetBase.Maker]]. This ensures that toolbars can always be dynamic and responsive.

## Properties

You can make a standalone toolbar and add widgets to it:

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

You can add any types of widgets to toolbars, although [[button]]s and [[func button]]s are the most common:

```Go
tb := core.NewToolbar(b)
tb.Maker(func(p *tree.Plan) {
    tree.Add(p, func(w *core.FuncButton) {
        w.SetFunc(core.SettingsWindow)
    })
    tree.Add(p, func(w *core.Switch) {
        w.SetText("Active")
    })
})
```

## Overflow

When you add more items to a toolbar than can fit on the screen, it places them in an overflow [[menu]]:

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

## Top bar

Toolbars are frequently added in [[doc:core.Body.AddTopBar]]:

```go
b.AddTopBar(func(bar *core.Frame) {
    core.NewToolbar(bar).Maker(func(p *tree.Plan) {
        tree.Add(p, func(w *core.Button) {
            w.SetText("Build")
        })
        tree.Add(p, func(w *core.Button) {
            w.SetText("Run")
        })
    })
})
```
