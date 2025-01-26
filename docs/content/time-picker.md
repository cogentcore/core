+++
Categories = ["Widgets"]
+++

A **time picker** is a [[widget]] that allows users to select an hour and minute.

Also see a [[date picker]] for selecting a year, month, and day.

## Properties

You can make a time picker and set its starting value:

```Go
core.NewTimePicker(b).SetTime(time.Now())
```

## Events

You can detect when a user [[events#change]]s the time:

```Go
tp := core.NewTimePicker(b).SetTime(time.Now())
tp.OnChange(func(e events.Event) {
    core.MessageSnackbar(tp, tp.Time.Format(core.SystemSettings.TimeFormat()))
})
```

## Time input

You can create a unified time input that allows users to select both a date and time using [[text field]]s and [[dialog]]s:

```Go
core.NewTimeInput(b).SetTime(time.Now())
```

You can hide the date or time part of a unified time input:

```Go
core.NewTimeInput(b).SetTime(time.Now()).SetDisplayDate(false)
```

You can detect when a user [[events#change]]s the value of a unified time input:

```Go
ti := core.NewTimeInput(b).SetTime(time.Now())
ti.OnChange(func(e events.Event) {
    core.MessageSnackbar(ti, ti.Time.Format("1/2/2006 "+core.SystemSettings.TimeFormat()))
})
```

## Duration input

You can also create a duration input that allows users to select a duration of time:

```Go
core.NewDurationInput(b).SetDuration(3 * time.Second)
```

You can detect when a user changes the value of a duration input:

```Go
di := core.NewDurationInput(b).SetDuration(3 * time.Second)
di.OnChange(func(e events.Event) {
    core.MessageSnackbar(di, di.Duration.String())
})
```
