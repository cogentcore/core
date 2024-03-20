# Meters

Cogent Core provides customizable meters for displaying bounded numeric values to users.

You can make a meter without any custom options:

```Go
gi.NewMeter(parent)
```

You can set the value of a meter:

```Go
gi.NewMeter(parent).SetValue(0.7)
```

You can set the minimum and maximum values of a meter:

```Go
gi.NewMeter(parent).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can make a meter render vertically:

```Go
gi.NewMeter(parent).Style(func(s *styles.Style) {
    s.Direction = styles.Column
})
```
