# Icons

Cogent Core provides more than 2,000 unique icons from the Material Symbols collection, allowing you to easily represent many things in a concise, visually pleasing, and language-independent way.

Icons are specified using their named constant in the [[icons]] package, and they are typically used in the context of another widget, like a button:

```Go
core.NewButton(parent).SetIcon(icons.Send)
```

However, you can also make a standalone icon widget:

```Go
core.NewIcon(parent).SetIcon(icons.Home)
```

You can convert an icon into its filled version:

```Go
core.NewButton(parent).SetIcon(icons.Home.Fill())
```

## Custom icons

You can add custom icons to your app using [[icons.AddFS]]:

```go
//go:embed icons/*.svg
var myIcons embed.FS

func main() { // or init()
    icons.AddFS(grr.Log1(fs.Sub(myIcons, "icons")))
}
```

Then, you can just use the string name of one of your icons, without the .svg extension, to specify your icon:

```go
core.NewButton(parent).SetIcon("my-icon-name")
```

Although only SVG files are supported for icons, you can easily embed a bitmap image file in an SVG file. Cogent Core provides an `svg` command line tool that can do this for you. To install it, run:

```sh
go install cogentcore.org/core/svg/cmd/svg@main
```

Then, to embed an image into an svg file, run:

```sh
svg embed-image my-image.png
```

This will create a file called `my-image.svg` that has the image embedded into it. Then, you can use that SVG file as an icon by adding the svg file to the icons filesystem, as described above.
