# Event handling in Goki

Goki provides a robust and easy-to-use event handling system that allows you to handle various events on widgets. To handle an event, simply call the `On[EventType]` method on any widget. For example:

```go
gi.NewButton(b).SetText("Click me").OnClick(func(e events.Event) {
    fmt.Println("Button clicked")
})
```

The `events.Event` object passed to the function can be used for various things like obtaining detailed event information or setting the event as handled to stop other event handlers from running. For example:

```go
gi.NewButton(b).SetText("Click me").OnClick(func(e events.Event) {
    fmt.Println("Button clicked at", e.Pos())
    e.SetHandled() // this event will not be handled by other event handlers now
})
```