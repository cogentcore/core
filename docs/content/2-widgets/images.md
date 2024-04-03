# Images

Cogent Core provides customizable images that allow you to render any image.

You should load images by embedding them so that they work across all platforms:

```go
//go:embed image.png
var myImage embed.FS
```

Then, you can open the image from your embedded filesystem:

```Go
gi.NewImage(parent).OpenFS(myImage, "image.png")
```

You can also open images directly from the operating system filesystem, but this is not recommended for app images you have in a specific location, since they may end up in a different location on different platforms:

```go
gi.NewImage(parent).Open("image.png")
```
