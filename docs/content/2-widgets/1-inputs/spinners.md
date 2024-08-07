Cogent Core provides customizable spinners, which are text fields designed for numeric input. They automatically support the parsing and validation of input, in addition to convenient incrementing and decrementing.

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

You can detect when the user changes the value of the spinner:

```Go
sp := core.NewSpinner(b)
sp.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Value changed to %g", sp.Value))
})
```