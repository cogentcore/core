+++
Categories = ["Widgets"]
+++

A **slider** is a [[widget]] for bounded numeric input. It provides more visual and less precise input than a [[spinner]].

## Properties

You can make a slider without any custom options:

```Go
core.NewSlider(b)
```

You can set the value of a slider:

```Go
core.NewSlider(b).SetValue(0.7)
```

You can set the minimum and maximum values of a slider:

```Go
core.NewSlider(b).SetMin(5.7).SetMax(18).SetValue(10.2)
```

You can set the amount that the arrow keys change the value by:

```Go
core.NewSlider(b).SetStep(0.2)
```

You can ensure that the value is always a multiple of the step:

```Go
core.NewSlider(b).SetStep(0.2).SetEnforceStep(true)
```

You can use an [[icon]] for the thumb of the slider:

```Go
core.NewSlider(b).SetIcon(icons.DeployedCodeFill)
```

## Events

You can detect when a user [[events#change|changes]] the value of a slider and then stops:

```Go
sr := core.NewSlider(b)
sr.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("OnChange: %v", sr.Value))
})
```

You can detect when a user changes the value of a slider as they slide ([[events#input|input]]):

```Go
sr := core.NewSlider(b)
sr.OnInput(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("OnInput: %v", sr.Value))
})
```
