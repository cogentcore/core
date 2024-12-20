**Events** contains explanations of common [[event]] types. You can also see the API documentation for an [exhaustive list](https://pkg.go.dev/cogentcore.org/core/events#Types) of event types.

## Mouse

Mouse events are triggered by the mouse/trackpad/touchpad on desktop platforms, and by finger motions on mobile platforms.

### Click

A click event is triggered when a user presses down and then up in a short period of time on a [[doc:styles/abilities.Clickable]] widget. Click events are often handled on [[button]]s:

```Go
core.NewButton(b).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Message sent")
})
```

### Double click

A double click event is triggered when a user [[#click]]s twice in a row in rapid succession on a [[doc:styles/abilities.DoubleClickable]] widget.

```Go
core.NewButton(b).SetText("Hello").OnDoubleClick(func(e events.Event) {
    core.MessageSnackbar(b, "Double clicked")
})
```

### Context menu

A context menu event is triggered when a user right clicks or [[#long press]]es on a widget. It results in any [[context menu]] for the widget being shown.

```Go
core.NewTextField(b).On(events.ContextMenu, func(e events.Event) {
    core.MessageSnackbar(b, "Context menu")
})
```

### Hover

A mouse enter event is triggered when a user moves their mouse over a widget. It sets the [[states#hovered|hovered]] state. Conversely, a mouse leave event is triggered when a user moves their mouse off of a widget. It clears the hovered state.

```Go
sr := core.NewSlider(b)
sr.On(events.MouseEnter, func(e events.Event) {
    core.MessageSnackbar(b, "Hovered")
})
sr.On(events.MouseLeave, func(e events.Event) {
    core.MessageSnackbar(b, "No longer hovered")
})
```

### Long hover

A long hover start event is triggered when a user leaves their mouse over a [[abilities#long hoverable|long hoverable]] widget for 500 milliseconds (that duration can be customized in the [[settings]]). That event results in any specified [[tooltip]] being shown. A long hover end event is sent whenever a user moves their mouse a certain distance, moves it off of a widget, or presses it down. That event causes any visible tooltip to disappear.

```Go
bt := core.NewButton(b).SetText("Hello")
bt.SetTooltip("You are long hovering/pressing")
bt.On(events.LongHoverStart, func(e events.Event) {
    core.MessageSnackbar(b, "Long hover start")
})
bt.On(events.LongHoverEnd, func(e events.Event) {
    core.MessageSnackbar(b, "Long hover end")
})
```

### Long press

Similar to a [[#long hover]] event, a long press start event is triggered when a user presses down on a [[abilities#long pressable|long pressable]] widget for 500 milliseconds. A long press event results in any specified [[tooltip]] being shown, and it also creates a [[#context menu]] event. A long press end event is triggered when a user moves their mouse/finger a certain distance, moves it off of a widget, or stops pressing down. That event causes any visible tooltip to disappear, but it does *not* hide any context menu created by the start event. Long press events are mainly relevant as a replacement for long hover events on mobile.

```Go
bt := core.NewButton(b).SetText("Hello")
bt.SetTooltip("You are long hovering/pressing")
bt.On(events.LongPressStart, func(e events.Event) {
    core.MessageSnackbar(b, "Long press start")
})
bt.On(events.LongPressEnd, func(e events.Event) {
    core.MessageSnackbar(b, "Long press end")
})
```

## Key

Key events are triggered by keyboard input through a physical or virtual keyboard. The main way to directly handle key events is the key chord event:

```Go
tf := core.NewTextField(b)
tf.OnKeyChord(func(e events.Event) {
    core.MessageSnackbar(b, string(e.KeyChord()))
})
```

### Key function

You can convert a key chord into a semantic key function. The key mappings are defined in the user's keyboard shortcut [[settings]]. Try pressing keyboard shortcuts in the text field below to see their semantic names (ex: `Ctrl+C`, `Ctrl+V`, `Ctrl+F`, `Ctrl+Enter`, `Ctrl+Left`, and `Right`, using `Cmd` instead of `Ctrl` on macOS).

```Go
tf := core.NewTextField(b)
tf.OnKeyChord(func(e events.Event) {
    core.MessageSnackbar(b, keymap.Of(e.KeyChord()).String())
})
```

### Shortcuts

[[Button]]s support keyboard shortcuts, which allow you to make a button trigger when a user presses a certain key chord or semantic key function, even if the button doesn't already have [[#focus]]. See [[button#events]] for more information.

## Generated

Generated events are created as a result of other events.

### Change

A change event is triggered after a user changes the value of a widget and then exits that widget to apply those changes. Change events are different from [[#input]] events, which happen as soon the user changes the value, even before it is applied. Change events are often handled on [[text field]]s:

```Go
tf := core.NewTextField(b)
tf.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, "OnChange: "+tf.Text())
})
```

Change events often cascade to higher-level elements. For example, change events for a widget in a [[list]] are also sent up to the list itself:

```Go
sl := []int{1, 3, 5}
core.NewList(b).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Slice: %v", sl))
})
```

### Input

An input event is triggered when a user changes the value of a widget, as soon as they make the change and before they apply it by exiting the element. Input events are different from [[#change]] events, which only happen once the changes are applied by exiting the element.

For example, a text field sends input events each time the user presses a key, even before they exit to apply the changes:

```Go
tf := core.NewTextField(b)
tf.OnInput(func(e events.Event) {
    core.MessageSnackbar(b, "OnInput: "+tf.Text())
})
```

Unlike [[#change]] events, input events rarely cascade to higher-level elements, so they must be handled directly on the relevant widget. Also, some widgets like [[chooser]]s support change events but not input events since they aren't applicable.

### Focus

A focus event is triggered when a [[doc:styles/abilities.Focusable]] widget gains keyboard focus. Conversely, a focus lost event is triggered when such a widget loses keyboard focus.

```Go
tf := core.NewTextField(b)
tf.OnFocus(func(e events.Event) {
    core.MessageSnackbar(b, "Focus gained")
})
tf.OnFocusLost(func(e events.Event) {
    core.MessageSnackbar(b, "Focus lost")
})
```
