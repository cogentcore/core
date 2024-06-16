# Date pickers

Cogent Core provides interactive date pickers that allow users to select a year, month, and day. See [time pickers](time-pickers) for time pickers that allow users to select an hour and minute, and a unified time input that allows users to input both a date and time.

You can make a date picker and set its starting value:

```Go
core.NewDatePicker(parent).SetTime(time.Now())
```
