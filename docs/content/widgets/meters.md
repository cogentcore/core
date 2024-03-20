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

You can make a meter render as a circle:

```Go
gi.NewMeter(parent).SetType(gi.MeterCircle)
```

You can make a meter render as a semicircle:

```Go
gi.NewMeter(parent).SetType(gi.MeterSemicircle)
```

You can add text to a circular meter:

```Go
gi.NewMeter(parent).SetType(gi.MeterCircle).SetText("50%")
```

You can add text to a semicircular meter:

```Go
gi.NewMeter(parent).SetType(gi.MeterSemicircle).SetText("50%")
```
