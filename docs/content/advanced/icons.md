# Custom icons

You can add custom icons to your app using icongen, a part of the `core generate` tool. Custom icons are typically placed in a `cicons` (custom icons) directory. In it, you can add all of your custom SVG icon files and an `icons.go` file with the following code:

```go
package cicons

//go:generate core generate -icons .
```

Then, once you run `go generate`, you can access your icons through your cicons package:

```go
core.NewButton(parent).SetIcon(cicons.MyIconName)
```

Although only SVG files are supported for icons, you can easily embed a bitmap image file in an SVG file. Cogent Core provides an `svg` command line tool that can do this for you. To install it, run:

```sh
go install cogentcore.org/core/svg/cmd/svg@main
```

Then, to embed an image into an svg file, run:

```sh
svg embed-image my-image.png
```

This will create a file called `my-image.svg` that has the image embedded into it. Then, you can use that SVG file as an icon as described above.
