+++
Categories = ["Widgets"]
+++

Cogent Core provides more than 2,000 unique **icons** from the [Material Symbols collection](https://fonts.google.com/icons), allowing you to easily represent many things in a concise, visually pleasing, and language-independent way. See [[icons]] for more info about this resource, and how to use additional icons in your app.

Icons are specified using their named variable in the [[doc:icons]] package, and they are typically used in the context of another [[widget]], like a [[button]]:

```Go
core.NewButton(b).SetIcon(icons.Send)
```

However, you can also make a standalone icon widget:

```Go
core.NewIcon(b).SetIcon(icons.Home)
```

You can use the filled version of an icon:

```Go
core.NewButton(b).SetIcon(icons.HomeFill)
```

## Styles

### Icon size

You can change the size of an icon:

```Go
ic := core.NewIcon(b).SetIcon(icons.Home)
ic.Styler(func(s *styles.Style) {
    s.IconSize.Set(units.Dp(40))
})
```

You can specify different icon sizes for each dimension:

```Go
ic := core.NewIcon(b).SetIcon(icons.Home)
ic.Styler(func(s *styles.Style) {
    s.IconSize.Set(units.Dp(40), units.Dp(20))
})
```

Icon size is an inherited property, so you can set it on a parent widget like a [[button]] and its icon will update accordingly:

```Go
bt := core.NewButton(b).SetText("Send").SetIcon(icons.Send)
bt.Styler(func(s *styles.Style) {
    s.IconSize.Set(units.Dp(30))
})
```

You can also use [[styles#font size]], which applies to all children including icons:

```Go
tf := core.NewTextField(b).SetText("Hello").SetLeadingIcon(icons.Euro).SetTrailingIcon(icons.OpenInNew)
tf.Styler(func(s *styles.Style) {
    s.Font.Size.Dp(20)
})
```


