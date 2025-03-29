+++
Categories = ["Concepts"]
+++

You can **animate** any [[widget]] by specifying an animation function, which is typically run at the refresh rate of the monitor.

The most commonly animated widget is a [[canvas]]:

```Go
t := float32(0)
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5*math32.Sin(t/500))
    pc.Fill.Color = colors.Scheme.Success.Base
    pc.PathDone()
})
c.Animate(func(a *core.Animation) {
    t += a.Dt
    c.NeedsRender()
})
```

## Pause

If you want to temporarily pause an animation, you can simply return early from your animation function:

```Go
pause := false
core.Bind(&pause, core.NewSwitch(b)).SetText("Pause")

t := float32(0)
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5*math32.Sin(t/500))
    pc.Fill.Color = colors.Scheme.Warn.Base
    pc.PathDone()
})
c.Animate(func(a *core.Animation) {
    if pause {
        return
    }
    t += a.Dt
    c.NeedsRender()
})
```

Also, animations associated with widgets that are currently not visible will automatically be paused.

## Stop

You can permanently stop an animation by setting the [[doc:core.Animation.Done]] field to true:

```Go
stop := false
core.NewButton(b).SetText("Stop").OnClick(func(e events.Event) {
    stop = true
})

t := float32(0)
c := core.NewCanvas(b).SetDraw(func(pc *paint.Painter) {
    pc.Circle(0.5, 0.5, 0.5*math32.Sin(t/500))
    pc.Fill.Color = colors.Scheme.Error.Base
    pc.PathDone()
})
c.Animate(func(a *core.Animation) {
    t += a.Dt
    c.NeedsRender()
    a.Done = stop
})
```

Also, animations associated with deleted widgets will automatically be permanently stopped.

## Other widgets

Any type of widget can be animated:

```Go
t := float32(0)
tx := core.NewText(b).SetText("0")
tx.Styler(func(s *styles.Style) {
	s.Min.X.Em(10)
})
tx.Animate(func(a *core.Animation) {
    t += a.Dt
    tx.SetText(fmt.Sprintf("%g", t))
    tx.UpdateRender()
})
```

Note: in that example, we use [[doc:core.WidgetBase.UpdateRender]] instead of [[doc:core.WidgetBase.Update]] to optimize performance. See [[update]] for more information. The [[styles#min]] setting is necessary to give the text enough space without redoing the layout every frame.

## Details

Note that, unlike for goroutines, the animation function is run in the main event loop and thus does *not* require any [[async]] protection. Using the animation API is better than using a goroutine since it automatically lines up with the app rendering timing, and it adapts to the screen refresh rate across platforms.

Using the [[doc:core.Animation.Delta]] field of the animation allows the animation to run at the same speed across refresh rates; faster refresh rates will lead to a smoother animation, but the overall speed will be the same.

If the animation is too intensive for the system to keep up, the animation rate will be automatically reduced, so it is not guaranteed to be exactly the refresh rate of the monitor. As such, unlike a goroutine, an animation should not cause any app hanging, even on web.
