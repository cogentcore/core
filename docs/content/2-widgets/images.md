# Images

Cogent Core provides customizable images that allow you to render any image.

You should load images by embedding them so that they work across all platforms:

```go
//go:embed image.png
var myImage embed.FS
```
