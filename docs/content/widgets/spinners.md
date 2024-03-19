# Spinners

Cogent Core provides customizable spinners, which are text fields designed for numeric input. They automatically support the parsing and validation of input.

You can make a spinner without any custom options:

```Go
gi.NewSpinner(parent)
```

You can set the starting value of a spinner:

```Go
gi.NewSpinner(parent).SetValue(12.7)
```
