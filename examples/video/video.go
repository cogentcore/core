package main

import (
	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/video"
)

func main() {
	b := core.NewBody("Basic Video Example")
	bx := core.NewFrame(b).Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
	})
	core.NewText(bx).SetText("video:").Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	v := video.NewVideo(bx)
	v.Styler(func(s *styles.Style) {
		s.Min.X.Px(200)
		s.Grow.Set(1, 1)
	})
	core.NewText(bx).SetText("filler:").Styler(func(s *styles.Style) {
		s.SetTextWrap(false)
	})
	core.NewText(b).SetText("footer:")
	errors.Log(v.Open("deer.mp4"))
	b.OnShow(func(e events.Event) {
		v.Play(0, 0)
	})
	b.RunMainWindow()
}
