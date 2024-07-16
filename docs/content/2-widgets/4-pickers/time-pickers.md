Cogent Core provides interactive time pickers that allow users to select an hour and minute. See [date pickers](date-pickers) for date pickers that allow users to select a year, month, and day.

You can make a time picker and set its starting value:

```Go
core.NewTimePicker(b).SetTime(time.Now())
```

You can detect when the user changes the time:

```Go
tp := core.NewTimePicker(b).SetTime(time.Now())
tp.OnChange(func(e events.Event) {
    core.MessageSnackbar(tp, tp.Time.Format(core.SystemSettings.TimeFormat()))
})
```

You can create a unified time input that allows users to select both a date and time using text fields and dialogs:

```Go
core.NewTimeInput(b).SetTime(time.Now())
```

You can detect when the user changes the value of a unified time input:

```Go
ti := core.NewTimeInput(b).SetTime(time.Now())
ti.OnChange(func(e events.Event) {
    core.MessageSnackbar(ti, ti.Time.Format("1/2/2006 "+core.SystemSettings.TimeFormat()))
})
```

You can also create a duration input that allows users to select a duration of time:

```Go
core.NewDurationInput(b).SetDuration(3 * time.Second)
```

You can detect when the user changes the value of a duration input:

```Go
di := core.NewDurationInput(b).SetDuration(3 * time.Second)
di.OnChange(func(e events.Event) {
    core.MessageSnackbar(di, di.Duration.String())
})
```
