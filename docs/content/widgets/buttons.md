# Buttons

Cogent Core provides interactive buttons that support a wide variety of different features.

Buttons can have text and icons:

```Go
gi.NewButton(parent).SetText("Download").SetIcon(icons.Download)
```

You can do something when a button is pressed using [[gi.WidgetBase.OnClick]]:

```Go
gi.NewButton(parent).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    gi.MessageSnackbar(parent, "Message sent")
})
```

Buttons can have popup menus associated with them:

```Go
gi.NewButton(parent).SetText("Share").SetIcon(icons.Share).SetMenu(func(m *gi.Scene) {
    gi.NewButton(m).SetText("Copy link")
    gi.NewButton(m).SetText("Send message")
})
```

## Types

Cogent Core provides several different types of buttons for different use cases.

Filled buttons are used for prominent actions, and they are the default type of button:

```Go
gi.NewButton(parent).SetType(gi.ButtonFilled).SetText("Filled")
```

Tonal buttons are similar to filled buttons but have less emphasis, making them appropriate for less important and more numerous actions:

```Go
gi.NewButton(parent).SetType(gi.ButtonTonal).SetText("Tonal")
```

Elevated buttons have a shadow and a light background, and they are typically used when a button needs to stand out from its surroundings, like if it is above an image:

```Go
gi.NewButton(parent).SetType(gi.ButtonElevated).SetText("Elevated")
```

Outlined buttons have a border and no background, and they are typically used for secondary actions like canceling or going back:

```Go
gi.NewButton(parent).SetType(gi.ButtonOutlined).SetText("Outlined")
```

Text buttons have no border and no background, and they should only be used for very low-importance buttons:

```Go
gi.NewButton(parent).SetType(gi.ButtonText).SetText("Text")
```

Action and menu buttons are the most minimal buttons, and they are typically only used in the context of other widgets, like toolbars and menus (buttons in toolbars will automatically become action buttons, and buttons in menus will automatically become menu buttons):

```Go
gi.NewButton(parent).SetType(gi.ButtonAction).SetText("Action")
```