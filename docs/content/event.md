+++
Categories = ["Concepts"]
+++

**Events** are user actions that you can process. To handle an event, simply call the `On{EventType}` method on any [[#widgets|widget]]. For example:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Button clicked")
})
```

The [[doc:events.Event]] object passed to the function can be used for things such as obtaining detailed event information. For example, you can determine the exact position of a click event:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprint("Button clicked at ", e.Pos()))
})
```
