package main

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/video"
)

func main() {
	b := gi.NewBody("Basic Video Example")
	v := video.NewVideo(b)
	v.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	grr.Log(v.Open("../videos/deer.mp4"))
	// grr.Log(v.Open("../videos/countdown.mp4"))
	w := b.NewWindow().Run()
	go v.Play(0, 0)
	w.Wait()
}
