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

## Event handling order

Event handlers are called in the reverse order that they are added (last added, first called), so that default event handling (added during initial widget configuration) can be overridden by special end-user handlers.

As with [[style]]rs, there are three levels of event handlers: `First`, `Normal`, and `Final`, which are called in that order (and last added, first processed within each). For example, this allows a default event handler to get the first crack at handling an event, even though it would be the last one called by default, by adding it to `First` using [[doc:core.WidgetBase.OnFirst]] instead.

If an event handler calls [[doc:events.Event.SetHandled]], further event handlers will not be called after it for that event.

There are two main categories of events that differ in the way they are handled: [[events#mouse]] events and [[events#key]] events.

### Mouse events

Mouse events are generally processed in a depth first manner, with the most deeply nested widget receiving the event first, and, if not yet handled, outer widgets get their chance.

### Basic vs processed events

Basic mouse events, which begin with `Mouse` (e.g., [[doc:events.MouseDown]], [[doc:events.MouseUp]], [[doc:events.MouseMove]]) are generally sent to all widgets, but are generally not handled. They are available if you need to do some kind of custom event processing.

Processed events such as [[events#click]] are generated from basic events (e.g., a Click is a Down and Up on the same Widget), and are the main target of event handlers.

### Key events and focus

The [[doc:events.KeyChord]] is the main [[events#key]] event, because it provides the full set of modifiers in a string-formatted form, whereas the more basic [[doc:events.Key]] event records each specific key down and up event using standardized key codes.

Key events are mainly sent to the single currently [[states#focused]] widget, which is determined by [[events#focus]] events and typically has a style indicating that it is in focus. However, see the next section for other ways to get key events based on priority ordering.

### Key event priority

[[events#Key]] events have special logic for priority ordering, so that outer containers can have a chance to process navigation and other such events, without being the focus widget. Specifically, the progressively higher parent widgets above the current focus widget with a [[doc:events.KeyChord]] First or Final handler get called before (First) and after (Final) the standard focus event handler.

Thus, if you need to intercept key events that might otherwise be processed by the focus widget, add a First KeyChord event handler function. If you only want to handle any events not otherwise processed by the focus widget (low priority), add a Final KeyChord event.  Generally, these container widgets should *not* have the [[abilities#focusable]] ability set, and they should *not* get standard focus events.

The builtin focus event handler calls all three priorities of handlers, so focus widgets can also take advantage of the priority ordering within themselves.
