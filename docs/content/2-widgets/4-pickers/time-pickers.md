# Time pickers

Cogent Core provides interactive time pickers that allow users to select an hour and minute. See [date pickers](date-pickers) for date pickers that allow users to select a year, month, and day.

You can make a time picker and set its starting value:

```Go
core.NewTimePicker(parent).SetTime(time.Now())
```
