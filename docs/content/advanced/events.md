# Events

## Low-level events

For low-level system events like [[events.MouseDown]] and [[events.KeyUp]], there is no `On{EventType}` helper method for listening to the event, so you must use [[gi.Widget.On]] with the event type as an argument.

## Event handling order

Event handlers are called in the reverse order that they are added (last added, first called), so that default event handling (added during initial widget configuration) can be overridden by special end-user handlers.

As with stylers, there are three levels of event handlers: `First`, regular, and `Final`, which are called in that order (and last added, first processed within each Listener). For example, this allows a default event handler to get the first crack at handling an event, even though it would be the last one called by default, by adding it to `First` using [[gi.WidgetBase.OnFirst]] instead.

If an event handler calls [[events.Event.SetHandled]], further event handlers will not be called after it for that event.

## Mouse and key events

There are two main categories of events that differ in the way they are handled: mouse/touch events and key events.

### Mouse events

Mouse events are generally processed in a _depth first_ manner, with the most deeply nested Widget receiving the event first, and, if not yet handled, outer Widgets get their chance.

### Basic vs. Processed events

Basic mouse events, which begin with `Mouse` (e.g., [[events.MouseDown]], [[events.MouseUp]] [[events.MouseMove]] are generally sent to all Widgets, but are generally not handled.  They are available if you need to do some kind of custom event processing.

Processed events such as [[events.Click]] are generated from basic events (e.g., a Click is a Down and Up on the same Widget), and are the main target of event handlers.

### Key events and Focus

The [[events.KeyChord]] is the main key event, because it provides the full set of modifiers in a string-formatted form, whereas the more basic [[events.Key]] event records each specific key down and up event using standardized key codes.

Key events are mainly sent to the single _current focus_ Widget, which is determined by focus-changing events and typically has a style indicating that it is in focus.  However, see the next section for other ways to get key events based on priority ordering.

### Key events

Key events ([[events.KeyChord]]) have special logic for priority ordering, so that outer containers can have a chance to process navigation and other such events, without being the Focus widget.  Specifically, the progressively higher Parent widgets above the current Focus widget with a [[events.KeyChord]] First or Final handler gets called before (First) and after (Final) the standard Focus event handler.

Thus, if you need to intercept key events that might otherwise be processed by the Focus Widget, add a First KeyChord event handler function.  If you only want to handle any events not otherwise processed by the Focus Widget (low priority), add a Final KeyChord event.  Generally, these container Widgets should not have the [[abilities.Focusable]] flag set, and they should not get standard Focus events.

The default Focus event handler calls all three priority ordering of handlers, so Focus Widgets can also take advantage of the priority ordering within themselves.
