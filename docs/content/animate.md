+++
Categories = ["Concepts"]
+++

You can **animate** any [[widget]] by specifying an animation function, which is run at the refresh rate of the monitor.

The most commonly animated widget is a [[canvas]]:

```Go
t := float32(0)
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5*math32.Sin(t))
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.PathDone()
})
c.Animate(func(a *core.Animation) {
    t += float32(a.Delta.Seconds())
    c.NeedsRender()
})
```

Note that, unlike for goroutines, the animation function is run in the main event loop and does *not* require any [[async]] protection. Using the animation API is better than using a goroutine since it automatically lines up with the app rendering timing, and it adapts to the screen refresh rate across platforms.

Using the `Delta` field of the animation allows the animation to run at the same speed across refresh rates; faster refresh rates will lead to a smoother animation, but the overall speed will be the same.
