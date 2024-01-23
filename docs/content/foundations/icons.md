# Icons

The [[icons]] package contains a number of SVG icons selected from [Material Design Symbols](https://fonts.google.com/icons), sourced through [marella/material-symbols](https://github.com/marella/material-symbols).

To specify a standard icon, it is best to refer to it by the named constants defined in the `icons` package, like the `icons.Send` case in the following example:

```Go
gi.NewButton(parent).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
    gi.MessageSnackbar(parent, "Message sent")
})
```

## Adding app-specific icons

To add your own icons, use something like the following cases.

If the icons are in a `icons` subdirectory, and you're building a `main` app:

```go
//go:embed icons/*.svg
var myIcons embed.FS

func main() {
    icons.AddFS(grr.Log1(fs.Sub(myIcons, "icons")))
}
```

Alternatively, if you have a separate icons directory in a larger, more complex app, you can do the embed directly in that directory, and include it in the main:

In `icons/icons.go`:

```go
//go:embed *.svg
var Icons embed.FS

func init() {
	icons.AddFS(Icons)
}
```

In a `main.go`, anonymously import the icons to trigger the init function:

```go
	_ "cogentcore.org/cogent/code/icons"
```

In either case, you can just use the string name, _without the .svg extension_, as an argument to any place where an icon is specified:

```go
    gi.NewButton(b).SetIcon("my_icon_name")
```    

## Using bitmap files instead of SVG

Although only SVG files are supported for icons, you can easily embed a bitmap image file in an SVG.  The `svg` tool can do this for you, as follows:

```sh
go install cogentcore.org/core/svg/cmd/svg@main
```

```sh
svg embed-image my-image.png
```

This will put create a file called `my-image.svg` that has the image embedded into it. Then, you can use that SVG file as an icon by adding the svg file to the icons filesystem, as described above.


