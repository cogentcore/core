# Asynchronous updating

Most of the time, updating happens synchronously through event handlers, stylers, updaters, and makers. However, sometimes you need to update content asynchronously from another goroutine. When you do so, you just need to protect any updates you make to widgets with [[core.WidgetBase.Async]].

For example, this code utilizes a goroutine to update text to the current time every second:

```Go
text := core.NewText(b)
text.Updater(func() {
    text.SetText(time.Now().Format("15:04:05"))
})
go func() {
    ticker := time.NewTicker(time.Second)
    for range ticker.C {
        text.Async(text.Update)
    }
}()
```

If you are calling multiple functions asynchronously, you should use [[core.WidgetBase.AsyncLock]] and [[core.WidgetBase.AsyncUnlock]] to surround the functions instead.
