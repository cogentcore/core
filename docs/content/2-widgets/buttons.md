# Buttons

Cogent Core provides interactive buttons that support a wide variety of different features.

You can make a button with text:

```Go
core.NewButton(parent).SetText("Download")
```

You can make a button with an icon:

```Go
core.NewButton(parent).SetIcon(icons.Download)
```

You can make a button with both text and an icon:

```Go
core.NewButton(parent).SetText("Download").SetIcon(icons.Download)
```

You can detect when a button is clicked:

```Go
core.NewButton(parent).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    core.MessageSnackbar(parent, "Message sent")
})
```

You can add a popup menu to a button:

```Go
core.NewButton(parent).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *core.Scene) {
    core.NewButton(m).SetText("Copy link")
    core.NewButton(m).SetText("Send message")
})
```

You can make a button trigger on a certain keyboard shortcut (`Command` is automatically converted to `Control` on non-macOS platforms):

```Go
core.NewButton(parent).SetText("Save").SetShortcut("Command+S").OnClick(func(e events.Event) {
    core.MessageSnackbar(parent, "File saved")
})
```

You can make a button trigger on a certain semantic key function, which have default bindings that the user can customize in their settings:

```Go
core.NewButton(parent).SetText("Open").SetKey(keyfun.Open).OnClick(func(e events.Event) {
    core.MessageSnackbar(parent, "File opened")
})
```

## Types

Cogent Core provides several different types of buttons for different use cases.

Filled buttons are used for prominent actions, and they are the default type of button:

```Go
core.NewButton(parent).SetType(core.ButtonFilled).SetText("Filled")
```

Tonal buttons are similar to filled buttons but have less emphasis, making them appropriate for less important and more numerous actions:

```Go
core.NewButton(parent).SetType(core.ButtonTonal).SetText("Tonal")
```

Elevated buttons have a shadow and a light background, and they are typically used when a button needs to stand out from its surroundings, like when it is above an image:

```Go
core.NewButton(parent).SetType(core.ButtonElevated).SetText("Elevated")
```

Outlined buttons have a border and no background, and they are typically used for secondary actions like canceling or going back:

```Go
core.NewButton(parent).SetType(core.ButtonOutlined).SetText("Outlined")
```

Text buttons have no border and no background, and they should only be used for low-importance actions:

```Go
core.NewButton(parent).SetType(core.ButtonText).SetText("Text")
```

Action and menu buttons are the most minimal buttons, and they are typically only used in the context of other widgets, like toolbars and menus (buttons in toolbars automatically become action buttons, and buttons in menus automatically become menu buttons):

```Go
core.NewButton(parent).SetType(core.ButtonAction).SetText("Action")
```