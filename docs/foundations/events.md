# Event handling in Cogent Core

Cogent Core provides a robust and easy-to-use event handling system that allows you to handle various events on widgets. To handle an event, simply call the `On[EventType]` method on any widget. For example:

<core-example>

```go
gi.NewButton(parent).SetText("Click me").OnClick(func(e events.Event) {
    fmt.Println("Button clicked")
})
```
</core-example>

The `events.Event` object passed to the function can be used for various things like obtaining detailed event information or setting the event as handled to stop other event handlers from running. For example:

<core-example>

```go
gi.NewButton(parent).SetText("Click me").OnClick(func(e events.Event) {
    fmt.Println("Button clicked at", e.Pos())
    e.SetHandled() // this event will not be handled by other event handlers now
})
```
</core-example>

Events are handled in the _reverse_ order in which they are added (Last added, first processed), so that default event handling (added during initial Widget configuration) can be overridden by special handlers.

There are two main categories of events that differ in the way they are handled: Mouse / Touch events, vs. Key events.

## Mouse events

Mouse events are generally processed in a _depth first_ manner, with the most deeply nested Widget receiving the event first, and, if not yet handled, outer Widgets get their chance.

### Basic vs. Processed events

Basic mouse events, which begin with `Mouse` (e.g., [[events.MouseButton]], [[events.MouseMove]] are generally sent to all Widgets, but are generally not handled.  They are available if you need to do some kind of custom event processing.

Processed events such as [[events.Click]] are generated from basic events (e.g., a Click is a Down and Up on the same Widget), and are the main target of event handlers.

## Key events and Focus

The [[events.KeyChord] is the main key event, because it provides the full set of modifiers in a string-formatted form, whereas the more basic [[events.Key]] event records each specific key down and up event using standardized key codes.

Key events are mainly sent to the single _current focus_ Widget, which is determined by focus-changing events and typically has a style indicating that it is in focus.  However, see the next section for other ways to get key events based on priority ordering.

## Priority Ordering (First, Final)

There are three levels of event handlers (see [[event.Listener]]): `First`, regular, and `Final`, which are processed in that order (and last added, first processed within each Listener).  This allows a default event handler to get first crack at handling a given event, even though it is added last, for example, by adding it to First.

### Key events

Key events ([[events.KeyChord]]) have special logic for priority ordering, so that outer containers can have a chance to process navigation and other such events, without being the Focus widget.  Specifically, the progressively higher Parent widgets above the current Focus widget with a [[events.KeyChord]] First or Final handler gets called before (First) and after (Final) the standard Focus event handler.

Thus, if you need to intercept key events that might otherwise be processed by the Focus Widget, add a First KeyChord event handler function.  If you only want to handle any events not otherwise processed by the Focus Widget (low priority), add a Final KeyChord event.  Generally, these container Widgets should not have the [[abilities.Focusable]] flag set, and they should not get standard Focus events.

The default Focus event handler calls all three priority ordering of handlers, so Focus Widgets can also take advantage of the priority ordering within themselves.



