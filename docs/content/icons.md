+++
Categories = ["Resources"]
+++

The [[doc:icons]] package contains the [Material Design Symbols](https://fonts.google.com/icons), sourced through [marella/material-symbols](https://github.com/marella/material-symbols).

See [[icon]] for information about the icon widget and how to set icons in stylers.

Icons are represented directly as an SVG string that draws the icon, and only those icons that your app actually uses are included, to minimize executable size. This also makes it easy to add new icon sets.

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

