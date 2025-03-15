+++
Categories = ["Widgets"]
+++

A **collapser** is a [[widget]] that can be collapsed or expanded by a user. It contains a summary part which is always visible, and a details part which is only visible when the collapser is expanded. Collapsers are similar to HTML's `&lt;details&gt;` and `&lt;summary&gt;` tags.

You can make a collapser:

```Go
cl := core.NewCollapser(b)
core.NewText(cl.Summary).SetText("Show details")
core.NewText(cl.Details).SetText("Long details about something")
```

You can put any [[widgets]] in a collapser:

```Go
cl := core.NewCollapser(b)
core.NewText(cl.Summary).SetType(core.TextHeadlineSmall).SetText("Widgets")
core.NewButton(cl.Details).SetText("Click me!")
core.NewSlider(cl.Details)
core.NewTextField(cl.Details)
```

Note that anything you enter into the input widgets above persists when you open and close the collapser; that is because collapsers don't delete the details, they just hide them.
