+++
Categories = ["Widgets"]
+++

A **body** is a [[frame]] widget that contains most of the elements constructed within a [[scene]].

The `b` widget that shows up in all of the examples in these docs represents the body that you typically add widgets to, as in the [[hello world tutorial]]:

<embed-page src="hello world tutorial" title="Hello world">

The [[doc:core.Body]] has methods to create different types of [[scene]] stages, such as the `RunMainWindow()` method in the above example, which manage the process of actually showing the body content to the user.

If you want to customize the behavior, do `b.NewWindow()` to get a new [[doc:core.Stage]], which has various optional parameters that can be configured.

