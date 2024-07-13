# Videos

Cogent Core provides interactive video players for playing video and audio media. Video support is currently experimental and only present on desktop platforms.

You can make a new video widget, open a video file, and play it:

```go
v := video.NewVideo(b)
errors.Log(v.Open("video.mp4"))
v.OnShow(func(e events.Event) {
    v.Play(0, 0)
})
```
