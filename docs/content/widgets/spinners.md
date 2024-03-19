# Spinners

Cogent Core provides customizable spinners, which are text fields designed for numeric input. They automatically support the parsing and validation of input, in addition to convenient incrementing and decrementing.

You can make a spinner without any custom options:

```Go
gi.NewSpinner(parent)
```

You can set the starting value of a spinner:

```Go
gi.NewSpinner(parent).SetValue(12.7)
```

You can set the minimum and maximum values of a spinner:

```Go
gi.NewSpinner(parent).SetMin(-0.5).SetMax(2.7)
```

You can set the amount that the plus and minus buttons and up and down arrow keys change the value by:

```Go
gi.NewSpinner(parent).SetStep(6)
```

You can ensure that the value is always a multiple of the step:

```Go
gi.NewSpinner(parent).SetStep(4).SetEnforceStep(true)
```
