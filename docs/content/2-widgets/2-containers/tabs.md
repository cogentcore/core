# Tabs

Cogent Core provides customizable tabs, which allow you to divide widgets into logical groups and give users the ability to freely navigate between them.

You can make tabs without any custom options:

```Go
ts := core.NewTabs(parent)
ts.NewTab("First")
ts.NewTab("Second")
```

You can add any widgets to tabs:

```Go
ts := core.NewTabs(parent)
first := ts.NewTab("First")
core.NewText(first).SetText("I am first!")
second := ts.NewTab("Second")
core.NewText(second).SetText("I am second!")
```

You can add as many tabs as you want:

```Go
ts := core.NewTabs(parent)
ts.NewTab("First")
ts.NewTab("Second")
ts.NewTab("Third")
ts.NewTab("Fourth")
```

You can add icons to tabs:

```Go
ts := core.NewTabs(parent)
ts.NewTab("First", icons.Home)
ts.NewTab("Second", icons.Explore)
```

You can make functional tabs, which can be closed and moved:

```Go
ts := core.NewTabs(parent).SetType(core.FunctionalTabs)
ts.NewTab("First")
ts.NewTab("Second")
ts.NewTab("Third")
```

You can make navigation tabs, which dynamically serve as a bottom navigation bar, side navigation rail, or side navigation drawer depending on the size of the screen:

```Go
ts := core.NewTabs(parent).SetType(core.NavigationAuto)
ts.NewTab("First", icons.Home)
ts.NewTab("Second", icons.Explore)
ts.NewTab("Third", icons.History)
```

You can allow the user to add new tabs:

```Go
ts := core.NewTabs(parent).SetNewTabButton(true)
ts.NewTab("First")
ts.NewTab("Second")
```
