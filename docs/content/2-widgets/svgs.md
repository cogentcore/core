# SVGs

Cogent Core provides customizable widgets that allow you to render any SVG object.

You should load SVG files by embedding them so that they work across all platforms:

```go
//go:embed icon.svg
var mySVG embed.FS
```

Then, you can open an SVG file from your embedded filesystem:

```Go
grr.Log(gi.NewSVG(parent).OpenFS(mySVG, "icon.svg"))
```
