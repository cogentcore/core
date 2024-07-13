# Meters

Cogent Core provides customizable meters for displaying bounded numeric values to users.

You can make a meter without any custom options:

```Go
core.NewMeter(b)
```

You can set the value of a meter:

```Go
core.NewMeter(b).SetValue(0.7)
```

You can set the minimum and maximum values of a meter:

```Go
core.NewMeter(b).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can make a meter render vertically:

```Go
core.NewMeter(b).Styler(func(s *styles.Style) {
    s.Direction = styles.Column
})
```

You can make a meter render as a circle:

```Go
core.NewMeter(b).SetType(core.MeterCircle)
```

You can make a meter render as a semicircle:

```Go
core.NewMeter(b).SetType(core.MeterSemicircle)
```

You can add text to a circular meter:

```Go
core.NewMeter(b).SetType(core.MeterCircle).SetText("50%")
```

You can add text to a semicircular meter:

```Go
core.NewMeter(b).SetType(core.MeterSemicircle).SetText("50%")
```
