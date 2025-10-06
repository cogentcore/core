+++
Categories = ["Widgets"]
Authors = ["Bea A. Author", "Test Ing Name"]
+++

A **button** is a [[widget]] that a user can click on to trigger a described action. See [[func button]] for a button [[value binding|bound]] to a function. There are various [[#types]] of buttons.

## Properties

You can make a button with text:

```Go
core.NewButton(b).SetText("Download")
```

You can make a button with an icon:

```Go
core.NewButton(b).SetIcon(icons.Download)
```

You can make a button with both text and an icon:

```Go
core.NewButton(b).SetText("Download").SetIcon(icons.Download)
```

## Events

You can detect when a button is clicked:

```Go
core.NewButton(b).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "Message sent")
})
```

You can make a button trigger on a certain keyboard shortcut (`Command` is automatically converted to `Control` on non-macOS platforms):

```Go
core.NewButton(b).SetText("Save").SetShortcut("Command+S").OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "File saved")
})
```

You can make a button trigger on a certain semantic key function, which have default bindings that the user can customize in their settings:

```Go
core.NewButton(b).SetText("Open").SetKey(keymap.Open).OnClick(func(e events.Event) {
    core.MessageSnackbar(b, "File opened")
})
```

## Types

There are several types of buttons for different use cases.

Filled buttons are used for prominent actions, and they are the default type of button:

```Go
core.NewButton(b).SetType(core.ButtonFilled).SetText("Filled")
```

Tonal buttons are similar to filled buttons but have less emphasis, making them appropriate for less important and more numerous actions:

```Go
core.NewButton(b).SetType(core.ButtonTonal).SetText("Tonal")
```

Elevated buttons have a shadow and a light background, and they are typically used when a button needs to stand out from its surroundings, like when it is above an image:

```Go
core.NewButton(b).SetType(core.ButtonElevated).SetText("Elevated")
```

Outlined buttons have a border and no background, and they are typically used for secondary actions like canceling or going back:

```Go
core.NewButton(b).SetType(core.ButtonOutlined).SetText("Outlined")
```

Text buttons have no border and no background, and they should only be used for low-importance actions:

```Go
core.NewButton(b).SetType(core.ButtonText).SetText("Text")
```

Action and menu buttons are the most minimal buttons, and they are typically only used in the context of other widgets, like toolbars and menus (buttons in toolbars automatically become action buttons, and buttons in menus automatically become menu buttons):

```Go
core.NewButton(b).SetType(core.ButtonAction).SetText("Action")
```
