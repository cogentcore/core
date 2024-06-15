# Sliders

Cogent Core provides customizable sliders for bounded numeric input.

You can make a slider without any custom options:

```Go
core.NewSlider(parent)
```

You can set the value of a slider:

```Go
core.NewSlider(parent).SetValue(0.7)
```

You can set the minimum and maximum values of a slider:

```Go
core.NewSlider(parent).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can set the amount that the arrow keys change the value by:

```Go
core.NewSlider(parent).SetStep(0.2)
```

You can ensure that the value is always a multiple of the step:

```Go
core.NewSlider(parent).SetStep(0.2).SetEnforceStep(true)
```

You can use an icon for the thumb of the slider:

```Go
core.NewSlider(parent).SetIcon(icons.DeployedCode.Fill())
```

You can detect when the user changes the value of the slider and then stops:

```Go
sr := core.NewSlider(parent)
sr.OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("OnChange: %v", sr.Value))
})
```

You can detect when the user changes the value of the slider as they slide it:

```Go
sr := core.NewSlider(parent)
sr.OnInput(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("OnInput: %v", sr.Value))
})
```
