# Styling

## Styling order

Stylers are called in the order that they are added (first added, first called), which means that the stylers added last get the final say on the styles. This means that the base stylers set during initial widget configuration will be overridden by special end-user stylers.

As with event handlers, there are three levels of stylers: `First`, regular, and `Final`, which are called in that order. For example, this allows you to set properties that affect stylers before they are called using [[gi.WidgetBase.StyleFirst]], like [[styles.Style.State]], and set style properties based on other style properties using [[gi.WidgetBase.StyleFinal]], like [[styles.Style.Min]] based on [[styles.Style.Direction]].

## Styling multiple widgets

You can style all widgets within a certain container at once using [[gi.Widget.OnWidgetAdded]]:

```Go
parent.OnWidgetAdded(func(w gi.Widget) {
    w.Style(func(s *styles.Style) {
        s.Color = colors.C(colors.Scheme.Error.Base)
    })
})
gi.NewLabel(parent).SetText("Label")
gi.NewSwitch(parent).SetText("Switch")
gi.NewTextField(parent).SetText("Text field")
```

You can style all widgets of a certain type within a certain container:

```Go
parent.OnWidgetAdded(func(w gi.Widget) {
    switch w := w.(type) {
    case *gi.Button:
        w.Style(func(s *styles.Style) {
            s.Border.Radius = styles.BorderRadiusSmall
        })
    }
})
gi.NewButton(parent).SetText("First")
gi.NewButton(parent).SetText("Second")
gi.NewButton(parent).SetText("Third")
```

You can style all widgets in the entire app using [[gi.App.SceneConfig]] in combination with [[gi.WidgetBase.OnWidgetAdded]]. For example, to make all buttons in your app have a small border radius, you can do the following:

```go
gi.TheApp.SetSceneConfig(func(sc *gi.Scene) {
    sc.OnWidgetAdded(func(w gi.Widget) {
        switch w := w.(type) {
        case *gi.Button:
            w.Style(func(s *styles.Style) {
                s.Border.Radius = styles.BorderRadiusSmall
            })
        }
    })
})
```
