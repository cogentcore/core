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

You can add as many tabs as you want:

```Go
ts := gi.NewTabs(parent)
ts.NewTab("First")
ts.NewTab("Second")
ts.NewTab("Third")
ts.NewTab("Fourth")
```

You can add icons to tabs:

```Go
ts := gi.NewTabs(parent)
ts.NewTab("First", icons.Home)
ts.NewTab("Second", icons.Explore)
```

You can make functional tabs, which can be closed and moved:

```Go
ts := gi.NewTabs(parent).SetType(gi.FunctionalTabs)
ts.NewTab("First")
ts.NewTab("Second")
ts.NewTab("Third")
```

You can add navigation tabs, which dynamically serve as a bottom navigation bar, side navigation rail, or side navigation drawer depending on the amount of space available:

```Go
ts := gi.NewTabs(parent).SetType(gi.NavigationAuto)
ts.NewTab("First")
ts.NewTab("Second")
ts.NewTab("Third")
```
