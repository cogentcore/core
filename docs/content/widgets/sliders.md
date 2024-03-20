# Sliders

Cogent Core provides customizable sliders for bounded numeric input.

You can make a slider without any custom options:

```Go
gi.NewSlider(parent)
```

You can set the starting value of a slider:

```Go
gi.NewSlider(parent).SetValue(0.7)
```

You can set the minimum and maximum values of a slider:

```Go
gi.NewSlider(parent).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can set the amount that the arrow keys change the value by:

```Go
gi.NewSlider(parent).SetStep(0.2)
```

You can ensure that the value is always a multiple of the step:

```Go
gi.NewSlider(parent).SetStep(0.2).SetEnforceStep(true)
```