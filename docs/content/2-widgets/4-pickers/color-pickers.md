# Color pickers

Cogent Core provides interactive color pickers that allow users to input colors using three sliders for the components of the HCT color system: hue, chroma, and tone. The tooltip for each slider contains more information about each component.

You can make a color picker and set its starting color to any color:

```Go
core.NewColorPicker(parent).SetColor(colors.Orange)
```

You can detect when the user changes the color:

```Go
cp := core.NewColorPicker(parent).SetColor(colors.Green)
cp.OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, colors.AsHex(cp.Color))
})
```
