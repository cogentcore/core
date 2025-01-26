+++
Categories = ["Widgets"]
+++

A **video** is a [[widget]] for playing video and audio media. Video support is currently experimental and only fully present on macOS and Linux. To use video players, you must first install these [dependencies](https://github.com/zergon321/reisen#dependencies).

For static images, use an [[image]] widget instead. For more programmatic control of rendering, use a [[canvas]]. For SVG, use an [[SVG]] widget.

You can make a new video widget, open a video file, and play it:

```go
v := video.NewVideo(b)
v.Styler(func(s *styles.Style) {
    s.Grow.Set(1, 1)
})
errors.Log(v.Open("video.mp4"))
v.OnShow(func(e events.Event) {
    v.Play(0, 0)
})
```

Here is a screenshot of the [video example](https://github.com/cogentcore/core/tree/main/examples/video):

![Screenshot of the video example](media/video.jpg)
