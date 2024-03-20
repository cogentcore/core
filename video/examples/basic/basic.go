package main

import (
	"cogentcore.org/core/gi"
	_ "cogentcore.org/core/giv"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/video"
)

func main() {
	b := gi.NewBody("Basic Video Example")
	bx := gi.NewLayout(b).Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	gi.NewLabel(bx).SetText("video:").Style(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	v := video.NewVideo(bx)
	v.Style(func(s *styles.Style) {
		s.Min.X.Px(200)
		s.Grow.Set(1, 1)
	})
	gi.NewLabel(bx).SetText("filler:").Style(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	gi.NewLabel(b).SetText("footer:")
	// grr.Log(v.Open("../videos/deer.mp4"))
	// grr.Log(v.Open("../videos/countdown.mp4"))
	grr.Log(v.Open("../videos/randy_first_360.mov")) // note: not uploaded -- good test case tho
	v.Rotation = -90
	w := b.NewWindow().Run()
	v.Play(0, 0)
	w.Wait()
}
