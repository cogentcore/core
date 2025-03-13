+++
Categories = ["Concepts"]
+++

You can **animate** any [[widget]] by specifying an animation function to be ran for every frame.

The most commonly animated widget is a [[canvas]]:

```Go
var t time.Duration
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.75*math32.Sin(float32(t.Seconds())))
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.PathDone()
})
c.Animate(func(a *core.Animation) {
    t += a.Delta
    c.NeedsRender()
})
```
