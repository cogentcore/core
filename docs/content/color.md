+++
Categories = ["Concepts"]
+++

**Colors** can be specified in several different ways. Colors are used for [[styles#color|styles]], and can be represented with the [[color picker]] widget.

## Color scheme

Colors should typically be specified using the color scheme, which automatically adapts to light/dark mode and is based on the theme color specified in the user [[settings]] or through [[doc:core.AppColor]]. The color scheme is based on Material Design 3 and uses their [HCT](https://material.io/blog/science-of-color-design) color format to ensure accessible color contrast.

Common scheme colors are explained below, with an interactive [[#color scheme demo]] below that. You can also see the API documentation for an exhaustive list of all [[doc:colors/matcolor.Scheme]] colors.

* Surface colors are relatively neutral colors often used for backgrounds and [[text]]
    * `Surface` is the basic background color
    * `OnSurface` is the color for text and other such things on top of backgrounds with `Surface` color
    * `SurfaceContainer` and other similar colors like `SurfaceContainerHigh` are for widgets that contrast some with the background, like [[text field]]s and [[dialog]]s
    * As for almost all scheme colors, there are `On` versions of `SurfaceContainer` colors, such as `OnSurfaceContainer`, which serves a similar purpose as `OnSurface`
* Accent colors are colorful colors used to convey something
    * All accent colors come with four versions:
        * `Base` for high-emphasis content
        * `On` for text/content on top of `Base`
        * `Container` for lower emphasis content
        * `OnContainer` for text/content on top of `Container`
    * The commonly used accents are:
        * `Primary` for important elements like filled [[button]]s
        * `Select` for selected elements
        * `Error` for error indicators or delete buttons
        * `Success` for success indicators
        * `Warn` for warnings

### Color scheme demo

Here is an interactive demo of all scheme colors. For example, try editing the first color below (primary base) and see the links on this page change to that color.

```Go
core.NewForm(b).SetStruct(colors.Scheme)
```

## Named colors

You can also use named colors:

```Go
bt := core.NewButton(b).SetText("Hello")
bt.Styler(func(s *styles.Style) {
    s.Background = colors.Uniform(colors.Orange)
})
```

Note that the color is wrapped in [[doc:colors.Uniform]]. That is because all colors in Cogent Core are specified as images, which allows for easy use of [[#gradient]]s and background [[#image]]s. For the [[#color scheme]] above, scheme colors are automatically converted to images, so you don't need to use colors.Uniform. You can use [[doc:colors.ToUniform]] to convert a scheme color back from an image to a color.

Named colors do not adjust to light/dark mode and user [[settings]], so you should use the [[#color scheme]] instead when possible. However, if you do need colors outside of the color scheme, you can use color normalization functions as explained below.

## Color normalization

You can use color normalization functions to make your colors adapt to the [[#color scheme]], ensuring sufficient contrast. Each color normalization function corresponds to a color scheme accent color version as documented above: `Base`, `On`, `Container`, and `OnContainer`.

For example, to make a color suitable for a high emphasis filled button, you can use [[doc:colors.ToBase]]:

```Go
bt := core.NewButton(b).SetText("Hello")
bt.Styler(func(s *styles.Style) {
    s.Background = colors.Uniform(colors.ToBase(colors.Orange))
    s.Color = colors.Uniform(colors.ToOn(colors.Orange))
})
```

## Gradient

You can specify a color as a gradient:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Background = gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
    s.Min.Set(units.Em(5))
})
```

You can make a radial gradient:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    s.Background = gradient.NewRadial().AddStop(colors.Purple, 0).AddStop(colors.Blue, 0.5).AddStop(colors.Skyblue, 1)
    s.Min.Set(units.Em(5))
})
```

You can rotate a gradient:

```Go
fr := core.NewFrame(b)
fr.Styler(func(s *styles.Style) {
    gr := gradient.NewLinear().AddStop(colors.Yellow, 0).AddStop(colors.Orange, 0.5).AddStop(colors.Red, 1)
    gr.SetTransform(math32.Rotate2D(math32.Pi/2))
    s.Background = gr
    s.Min.Set(units.Em(5))
})
```

## Image

You can use any image as a color, including all those supported by the [[image]] widget.
