+++
Categories = ["Concepts"]
+++

Most of the time, updating happens synchronously through [[event]] handlers, [[style]]rs, [[update]]rs, and [[plan|makers]]. However, sometimes you need to update content **asynchronously** from another goroutine. When you do so, you just need to protect any updates you make to [[widget]]s with [[doc:core.WidgetBase.AsyncLock]] and [[doc:core.WidgetBase.AsyncUnlock]].

For example, this code utilizes a goroutine to update [[text]] to the current time every second:

```Go
text := core.NewText(b)
text.Updater(func() {
    text.SetText(time.Now().Format("15:04:05"))
})
go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        text.AsyncLock()
        text.Update()
        text.AsyncUnlock()
    }
}()
```
