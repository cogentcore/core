**Styles** contains explanations of common [[style]] properties. You can also see the API documentation for an [exhaustive list](https://pkg.go.dev/cogentcore.org/core/styles#Style) of style properties. You can experiment with style properties in the [[style playground]].

## Color

Many style properties involve colors, which can be specified in several ways.

### Color scheme

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

Here is an interactive demo of all scheme colors. Try editing the first color below (primary base) and see the links on this page change to that color.

```Go
core.NewForm(b).SetStruct(colors.Scheme)
```
