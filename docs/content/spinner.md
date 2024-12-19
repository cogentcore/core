+++
Categories = ["Widgets"]
+++

A **spinner** is a [[text field]] designed for numeric input. It automatically supports the parsing and validation of input, in addition to convenient incrementing and decrementing. It provides more precise and less visual input than a [[slider]].

## Properties

You can make a spinner without any custom options:

```Go
core.NewSpinner(b)
```

You can set the value of a spinner:

```Go
core.NewSpinner(b).SetValue(12.7)
```

You can set the minimum and maximum values of a spinner:

```Go
core.NewSpinner(b).SetMin(-0.5).SetMax(2.7)
```

You can set the amount that the plus and minus buttons and up and down arrow keys change the value by:

```Go
core.NewSpinner(b).SetStep(6)
```

You can ensure that the value is always a multiple of the step:

```Go
core.NewSpinner(b).SetStep(4).SetEnforceStep(true)
```

You can make a spinner outlined instead of filled:

```Go
core.NewSpinner(b).SetType(core.TextFieldOutlined)
```

You can change the way that the value is formatted:

```Go
core.NewSpinner(b).SetFormat("%X").SetStep(1).SetValue(44)
```

## Events

You can detect when a user [[event#change|changes]] the value of a spinner:

```Go
sp := core.NewSpinner(b)
sp.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Value changed to %g", sp.Value))
})
```
