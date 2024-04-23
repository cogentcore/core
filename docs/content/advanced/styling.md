# Styling

## Styling order

Stylers are called in the order that they are added (first added, first called), which means that the stylers added last get the final say on the styles. This means that the base stylers set during initial widget configuration will be overridden by special end-user stylers.

As with event handlers, there are three levels of stylers: `First`, regular, and `Final`, which are called in that order. For example, this allows you to set properties that affect stylers before they are called using [[core.WidgetBase.StyleFirst]], like [[styles.Style.State]], and set style properties based on other style properties using [[core.WidgetBase.StyleFinal]], like [[styles.Style.Min]] based on [[styles.Style.Direction]].

## Styling multiple widgets

You can style all widgets within a certain container at once using [[core.Widget.OnWidgetAdded]]:

```Go
fr := core.NewFrame(parent)
fr.OnWidgetAdded(func(w core.Widget) {
    w.Style(func(s *styles.Style) {
        s.Color = colors.C(colors.Scheme.Error.Base)
    })
})
core.NewText(fr).SetText("Label")
core.NewSwitch(fr).SetText("Switch")
core.NewTextField(fr).SetText("Text field")
```

You can style all widgets of a certain type within a certain container:

```Go
fr := core.NewFrame(parent)
fr.OnWidgetAdded(func(w core.Widget) {
    switch w := w.(type) {
    case *core.Button:
        w.Style(func(s *styles.Style) {
            s.Border.Radius = styles.BorderRadiusSmall
        })
    }
})
core.NewButton(fr).SetText("First")
core.NewButton(fr).SetText("Second")
core.NewButton(fr).SetText("Third")
```

You can style all widgets in the entire app using [[core.App.SceneConfig]] in combination with [[core.WidgetBase.OnWidgetAdded]]. For example, to make all buttons in your app have a small border radius, you can do the following:

```go
core.TheApp.SetSceneConfig(func(sc *core.Scene) {
    sc.OnWidgetAdded(func(w core.Widget) {
        switch w := w.(type) {
        case *core.Button:
            w.Style(func(s *styles.Style) {
                s.Border.Radius = styles.BorderRadiusSmall
            })
        }
    })
})
```
