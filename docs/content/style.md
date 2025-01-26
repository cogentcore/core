+++
Categories = ["Concepts"]
+++

**Styling** allows you to easily customize the appearance of [[widget]]s at any level. See [[styles]] for explanations of common style properties. You can experiment with style properties in the [[style playground]].

You can change any style properties of a widget:

```Go
core.NewText(b).SetText("Bold text").Styler(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can change the [[color]]s of a widget using Cogent Core's dynamic [[color#color scheme]] system:

```Go
core.NewButton(b).SetText("Success button").Styler(func(s *styles.Style) {
    s.Background = colors.Scheme.Success.Base
    s.Color = colors.Scheme.Success.On
})
```

You can change the [[styles#size]] of a widget using Cogent Core's flexible [[unit]] system:

```Go
core.NewFrame(b).Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(50))
    s.Background = colors.Scheme.Primary.Base
})
```

## Abilities and states

[[Abilities]] are set in stylers, and you can set styles based on [[states]]. See those pages for more information.

## Styling order

Stylers are called in the order that they are added (first added, first called), which means that the stylers added last get the final say on the styles. This means that the base stylers set during initial widget configuration will be overridden by special end-user stylers.

As with [[event]] handlers, there are three levels of stylers: `First`, `Normal`, and `Final`, which are called in that order. For example, this allows you to set properties that affect stylers before they are called using [[doc:core.WidgetBase.FirstStyler]], like [[state]]s, and set style properties based on other style properties using [[doc:core.WidgetBase.FinalStyler]], like [[styles#min]] based on [[styles#direction]].

## Style multiple widgets

You can style all direct children of a container at once using [[doc:tree.NodeBase.OnChildAdded]]:

```Go
fr := core.NewFrame(b)
fr.SetOnChildAdded(func(n tree.Node) {
    core.AsWidget(n).Styler(func(s *styles.Style) {
        s.Color = colors.Scheme.Error.Base
    })
})
core.NewText(fr).SetText("Label")
core.NewSwitch(fr).SetText("Switch")
core.NewTextField(fr).SetText("Text field")
```

You can style all direct children of a certain type in a container:

```Go
fr := core.NewFrame(b)
fr.SetOnChildAdded(func(n tree.Node) {
    switch n := n.(type) {
    case *core.Button:
        n.Styler(func(s *styles.Style) {
            s.Border.Radius = styles.BorderRadiusSmall
        })
    }
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

### Style all widgets

You can style all widgets in the entire app using [[doc:core.App.SceneInit]] in conjunction with [[doc:core.Scene.WidgetInit]]. For example, to make all [[button]]s in your app have a small [[styles#border radius]], you can do the following:

```go
core.TheApp.SetSceneInit(func(sc *core.Scene) {
    sc.SetWidgetInit(func(w core.Widget) {
        switch w := w.(type) {
        case *core.Button:
            w.Styler(func(s *styles.Style) {
                s.Border.Radius = styles.BorderRadiusSmall
            })
        }
    })
})
```
