+++
Categories = ["Concepts"]
+++

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

A mouse enter event is triggered when a user moves their mouse over a widget. It sets the [[states#hovered]] state. Conversely, a mouse leave event is triggered when a user moves their mouse off of a widget. It clears the hovered state.

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

A long hover start event is triggered when a user leaves their mouse over a [[abilities#long hoverable]] widget for 250 milliseconds (that duration can be customized in the [[settings]]). That event results in any specified [[tooltip]] being shown. A long hover end event is sent whenever a user moves their mouse a certain distance, moves it off of a widget, or presses it down. That event causes any visible tooltip to disappear.

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

Similar to a [[#long hover]] event, a long press start event is triggered when a user presses down on a [[abilities#long pressable]] widget for 500 milliseconds. On mobile platforms only, a long press event results in any specified [[tooltip]] being shown, and it also creates a [[#context menu]] event (on mobile only). A long press end event is triggered when a user moves their mouse/finger a certain distance, moves it off of a widget, or stops pressing down. That event causes any visible tooltip to disappear, but it does *not* hide any context menu created by the start event. Long press events are mainly relevant as a replacement for long hover events on mobile.

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

### Change on input

You can make a widget send a [[#change]] event when it receives an [[#input]] event by calling `SendChangeOnInput`:

```Go
tf := core.NewTextField(b)
tf.SendChangeOnInput()
tf.OnChange(func(e events.Event) {
    core.MessageSnackbar(b, "OnChange: "+tf.Text())
})
```

This can be useful for things such as [[bind|value binding]], allowing you to have a bound variable update on input events.

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

#### Set focus

You can programmatically give a widget focus with [[doc:core.WidgetBase.SetFocus]]:

```Go
bt := core.NewButton(b).SetText("Focus")
tf := core.NewTextField(b)
bt.OnFinal(events.Click, func(e events.Event) {
    tf.SetFocus()
})
```

SetFocus only fully works for widgets that have already been shown. For newly created widgets, consider using [[doc:core.WidgetBase.StartFocus]] (documented right below), or [[#defer]] your SetFocus call.

You can make a widget have starting focus (so that it gets focus when there is a [[#show]] event):

```Go
pg := core.NewPages(b)
pg.AddPage("home", func(pg *core.Pages) {
    core.NewButton(pg).SetText("Next").OnClick(func(e events.Event) {
        pg.Open("next")
    })
})
pg.AddPage("next", func(pg *core.Pages) {
    tf := core.NewTextField(pg)
    tf.StartFocus()
})
```

### Show

A show event is triggered when the [[scene]] containing a widget is first shown to a user. It is also sent whenever a major content managing widget such as [[tabs]] or [[pages]] shows a new tab/page/element (it is sent using [[doc:core.WidgetBase.Shown]]). It is often used to do things that only work once everything is configured and visible, or for expensive actions that should only happen when truly necessary.

```Go
ts := core.NewTabs(b)
home := ts.NewTab("Home")
data := ts.NewTab("Data")
tx := core.NewText(data)
tx.OnShow(func(e events.Event) {
    // Pretend this is some expensive computation that only needs to happen once this tab is visible
    tx.SetText("Result of expensive computation: "+time.Now().Format(time.DateTime)).Update()
})
```

### Close

A close event is triggered when the [[scene]] containing a widget is about to close. It is often used to save unsaved edits or apply changes. You can also consider a [[dialog#close dialog]], which builds on close events.

```Go
bt := core.NewButton(b).SetText("Open dialog")
bt.OnClick(func(e events.Event) {
    d := core.NewBody("Dialog")
    d.OnClose(func(e events.Event) {
        core.MessageSnackbar(bt, "Dialog closed")
    })
    d.AddOKOnly().RunDialog(bt)
})
```

### Defer

Although not technically an event, deferring is related to event handling. Deferring allows you to add a function that will be called after the next [[scene]] update/render and then removed. This is often necessary when you make structural changes to widgets and then want to do a follow-up action after the layout and rendering is done.

For example, in the code below, using [[doc:core.WidgetBase.Defer]] for [[#set focus]] is necessary, as the text field cannot be properly focused until it has been styled and the layout has been updated by the [[update]] call.

```Go
bt := core.NewButton(b).SetText("Create text field")
bt.OnClick(func(e events.Event) {
    tf := core.NewTextField(b)
    tf.Defer(func() {
        tf.SetFocus()
    })
    b.Update()
})
```
