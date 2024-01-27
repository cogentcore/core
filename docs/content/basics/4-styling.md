# Styling

(TODO)

## Basics

## OnWidgetAdded

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
