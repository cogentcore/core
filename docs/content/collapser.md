+++
Categories = ["Widgets"]
+++

A **collapser** is a [[widget]] that can be collapsed or expanded by a user. It contains a summary part which is always visible, and a details part which is only visible when the collapser is expanded. Collapsers are similar to HTML's `<details>` and `<summary>` tags.

You can make a collapser:

```Go
cl := core.NewCollapser(b)
core.NewText(cl.Summary).SetText("Show details")
core.NewText(cl.Details).SetText("Long details about something")
```
