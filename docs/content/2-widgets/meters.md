# Meters

Cogent Core provides customizable meters for displaying bounded numeric values to users.

You can make a meter without any custom options:

```Go
core.NewMeter(parent)
```

You can set the value of a meter:

```Go
core.NewMeter(parent).SetValue(0.7)
```

You can set the minimum and maximum values of a meter:

```Go
core.NewMeter(parent).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can make a meter render vertically:

```Go
core.NewMeter(parent).Style(func(s *styles.Style) {
    s.Direction = styles.Column
})
```

You can make a meter render as a circle:

```Go
core.NewMeter(parent).SetType(core.MeterCircle)
```

You can make a meter render as a semicircle:

```Go
core.NewMeter(parent).SetType(core.MeterSemicircle)
```

You can add text to a circular meter:

```Go
core.NewMeter(parent).SetType(core.MeterCircle).SetText("50%")
```

You can add text to a semicircular meter:

```Go
core.NewMeter(parent).SetType(core.MeterSemicircle).SetText("50%")
```
