+++
Categories = ["Widgets"]
+++

A **date picker** is a [[widget]] that allows users to select a year, month, and day.

Also see a [[time picker]] that allows users to select an hour and minute, a unified [[time picker#time input]] that allows users to select both a date and time using [[text field]]s and [[dialog]]s, and a [[time picker#duration input]].

## Properties

You can make a date picker and set its starting value:

```Go
core.NewDatePicker(b).SetTime(time.Now())
```

## Events

You can detect when a user [[events#change]]s the date:

```Go
dp := core.NewDatePicker(b).SetTime(time.Now())
dp.OnChange(func(e events.Event) {
    core.MessageSnackbar(dp, dp.Time.Format("1/2/2006"))
})
```
