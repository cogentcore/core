+++
Categories = ["Widgets"]
+++

A **color picker** is a [[widget]] that allows users to input [[color]]s using three [[slider]]s for the components of the HCT color system: hue, chroma, and tone. The [[tooltip]] for each slider contains more information about each component.

## Properties

You can make a color picker and set its starting color to any color:

```Go
core.NewColorPicker(b).SetColor(colors.Orange)
```

## Events

You can detect when a user [[events#change]]s the color:

```Go
cp := core.NewColorPicker(b).SetColor(colors.Green)
cp.OnChange(func(e events.Event) {
    core.MessageSnackbar(cp, colors.AsHex(cp.Color))
})
```

## Color button

You can make a [[button]] that opens a color picker [[dialog]]:

```Go
core.NewColorButton(b).SetColor(colors.Purple)
```

You can detect when a user [[events#change]]s the color using the dialog:

```Go
cb := core.NewColorButton(b).SetColor(colors.Gold)
cb.OnChange(func(e events.Event) {
    core.MessageSnackbar(cb, colors.AsHex(cb.Color))
})
```
