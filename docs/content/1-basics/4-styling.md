# Styling

Cogent Core provides a highly versatile styling system that allows you to easily customize every aspect of the appearance of widgets at any level.

You can change styling properties of a specific widget:

```Go
gi.NewLabel(parent).SetText("Bold text").Style(func(s *styles.Style) {
    s.Font.Weight = styles.WeightBold
})
```

You can use Cogent Core's color scheme system based on Material Design 3's dynamic color to change the colors of a widget:

```Go
gi.NewButton(parent).SetText("Success button").Style(func(s *styles.Style) {
    s.Background = colors.C(colors.Scheme.Success.Base)
    s.Color = colors.C(colors.Scheme.Success.On)
})
```

You can use Cogent Core's flexible unit system to specify sizing properties of a widget in one of many different units. The most common units are `dp` (density-independent pixels, or 1/160th of 1 inch), and `em` (the font size of the element).

```Go
gi.NewLabel(parent).SetText("Big text").Style(func(s *styles.Style) {
    s.Font.Size.Dp(21)
})
```

Throughout the documentation for different widgets, you will learn how to use various other styling properties. To see all styling properties available, you can look at the documentation for [[styles.Style]].

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
