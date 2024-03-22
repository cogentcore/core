# Tabs

Cogent Core provides customizable tabs, which allow you to divide widgets into logical groups and give users the ability to freely navigate between them.

You can make tabs without any custom options:

```Go
ts := gi.NewTabs(parent)
ts.NewTab("First")
ts.NewTab("Second")
```