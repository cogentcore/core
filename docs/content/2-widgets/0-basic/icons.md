# Icons

Cogent Core provides more than 2,000 unique icons from the Material Symbols collection, allowing you to easily represent many things in a concise, visually pleasing, and language-independent way.

Icons are specified using their named variable in the [[icons]] package, and they are typically used in the context of another widget, like a button:

```Go
core.NewButton(parent).SetIcon(icons.Send)
```

However, you can also make a standalone icon widget:

```Go
core.NewIcon(parent).SetIcon(icons.Home)
```

You can use the filled version of an icon:

```Go
core.NewButton(parent).SetIcon(icons.HomeFill)
```

You can see the [advanced icons page](../../advanced/icons) for information about custom icons if you need it.
