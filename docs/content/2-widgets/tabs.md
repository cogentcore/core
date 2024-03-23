# Tabs

Cogent Core provides customizable tabs, which allow you to divide widgets into logical groups and give users the ability to freely navigate between them.

You can make tabs without any custom options:

```Go
ts := gi.NewTabs(parent)
ts.NewTab("First")
ts.NewTab("Second")
```

You can add any widgets to tabs:

```Go
ts := gi.NewTabs(parent)
first := ts.NewTab("First")
gi.NewLabel(first).SetText("I am first!")
second := ts.NewTab("Second")
gi.NewLabel(second).SetText("I am second!")
```
