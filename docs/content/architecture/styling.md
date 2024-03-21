# Styling

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
