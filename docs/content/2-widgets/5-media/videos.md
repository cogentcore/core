Cogent Core provides interactive video players for playing video and audio media. Video support is currently experimental and only fully present on macOS and Linux. To use video players, you must first install these [dependencies](https://github.com/zergon321/reisen#dependencies).

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

![Screenshot of the Cogent Core video example](videos.jpg)
