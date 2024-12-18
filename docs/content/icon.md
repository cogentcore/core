+++
Categories = ["Widgets"]
+++

Cogent Core provides more than 2,000 unique **icons** from the [Material Symbols collection](https://fonts.google.com/icons), allowing you to easily represent many things in a concise, visually pleasing, and language-independent way.

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

## Custom icons

You can add custom icons to your app using icongen, a part of the [[generate]] tool. Custom icons are typically placed in a `cicons` (custom icons) directory. In it, you can add all of your custom SVG icon files and an `icons.go` file with the following code:

```go
package cicons

//go:generate core generate -icons .
```

Then, once you run `go generate`, you can access your icons through your cicons package, where icon names are automatically transformed into CamelCase:

```go
core.NewButton(b).SetIcon(cicons.MyIconName)
```

### Image icons

Although only SVG files are supported for icons, you can easily embed a bitmap image file in an SVG file. Cogent Core provides an `svg` command line tool that can do this for you. To install it, run:

```sh
go install cogentcore.org/core/svg/cmd/svg@main
```

Then, to embed an image into an svg file, run:

```sh
svg embed-image my-image.png
```

This will create a file called `my-image.svg` that has the image embedded into it. Then, you can use that SVG file as an icon as described above.
