# Date pickers

Cogent Core provides interactive date pickers that allow users to select a year, month, and day. See [time pickers](time-pickers) for time pickers that allow users to select an hour and minute, a unified time input that allows users to select both a date and time using text fields and dialogs, and a duration input.

You can make a date picker and set its starting value:

```Go
core.NewDatePicker(parent).SetTime(time.Now())
```

You can detect when the user changes the date:

```Go
dp := core.NewDatePicker(parent).SetTime(time.Now())
dp.OnChange(func(e events.Event) {
    core.MessageSnackbar(dp, dp.Time.Format("1/2/2006"))
})
```
