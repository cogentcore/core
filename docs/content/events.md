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

For example, a slider sends input events as the user slides it, even before they let go to apply the changes:

```Go
sr := core.NewSlider(b)
sr.OnInput(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("OnInput: %v", sr.Value))
})
```

Unlike [[#change]] events, input events rarely cascade to higher-level elements, so they must be handled directly on the relevant widget. Also, some widgets like [[chooser]]s support change events but not input events since they aren't applicable.
