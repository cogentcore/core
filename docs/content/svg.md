+++
Name = "SVG"
Categories = ["Widgets"]
+++

An **SVG** [[widget]] renders SVG data.

If you have image files instead of SVG files, use an [[image]] widget. For more programmatic control of rendering, you can use a [[canvas]]. For rendering HTML, see [[HTML]].

## Properties

You should load SVG files by embedding them so that they work across all platforms:

```go
//go:embed icon.svg
var mySVG embed.FS
```

Then, you can open an SVG file from your embedded filesystem:

```Go
errors.Log(core.NewSVG(b).OpenFS(mySVG, "icon.svg"))
```

You can change the size of an SVG:

```Go
svg := core.NewSVG(b)
errors.Log(svg.OpenFS(mySVG, "icon.svg"))
svg.Styler(func(s *styles.Style) {
    s.Min.Set(units.Dp(128))
})
```

You can make it so that users can pan and zoom the SVG:

```Go
svg := core.NewSVG(b)
svg.SetReadOnly(false)
errors.Log(svg.OpenFS(mySVG, "icon.svg"))
```

You can directly set an SVG from an SVG data string:

```Go
errors.Log(core.NewSVG(b).ReadString(`<rect width="100" height="100" fill="red"/>`))
```

You can also open SVGs directly from the system filesystem, but this is not recommended for SVGs built into your app, since they will end up in a different location on different platforms:

```go
errors.Log(core.NewSVG(b).Open("icon.svg"))
```
