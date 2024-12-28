+++
Categories = ["Concepts"]
+++

An **event** is a user action that you can process. See [[events]] for explanations of common event types.

To handle an event, simply call the `On{EventType}` method on any [[widget]]. For example:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Button clicked")
})
```

The [[doc:events.Event]] object passed to the function can be used for things such as obtaining detailed event information. For example, you can determine the exact position of a [[events#click]] event:

```Go
core.NewButton(b).SetText("Click me!").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprint("Button clicked at ", e.Pos()))
})
```

## Abilities and states

[[Abilities]] determine the events that a widget can receive, and [[states]] are set as a result of events. Therefore, events act as a bridge between abilities and corresponding states.

For example, a [[abilities#hoverable]] ability allows for [[events#hover]] events, which set the [[states#hovered]] state.

If events aren't working as expected, the reason might be that the right abilities aren't set.

## Low-level events

For some lower-level events like [[doc:events.MouseDown]] and [[doc:events.KeyUp]], there is no `On{EventType}` helper method for listening to the event, so you must use [[doc:core.WidgetBase.On]] with the event type as an argument. For example:

```Go
core.NewButton(b).SetText("Press").On(events.MouseDown, func(e events.Event) {
    core.MessageSnackbar(b, "Mouse down")
})
```
