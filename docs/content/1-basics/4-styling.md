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
gi.NewButton(parent).SetText("Error button").Style(func(s *styles.Style) {
    s.Background = colors.C(colors.Scheme.Error.Base)
    s.Color = colors.C(colors.Scheme.Error.On)
})
```

You can use Cogent Core's flexible unit system to specify sizing properties of a widget in one of many different units. The most common units are `dp` (density-independent pixels, or 1/160th of 1 inch), and `em` (the font size of the element).

```Go
gi.NewLabel(parent).SetText("Big text").Style(func(s *styles.Style) {
    s.Font.Size.Dp(21)
})
```

## Global configuration functions

If you want to specify default styling or other configuration parameters for all widgets in an app, you can use the [[gi.App.SceneConfig]] field in combination with [[gi.WidgetBase.OnWidgetAdded]]. For example, to make all buttons have a small border radius, you could do the following:

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
