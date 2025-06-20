+++
Categories = ["Concepts"]
+++

This page contains a list of fundamental **principles** that Cogent Core is built on. There are occasional exceptions to these principles, but they should be followed almost all of the time, and there must be a very clear reason for not following them.

## Code is always the best solution

A common approach to programming, especially for web, is to reduce the amount of actual programming that takes place and instead use simplified markup, configuration, and selection formats like CSS, HTML, YAML, and Regex. The promise of this is that you will be able to accomplish everything you need to do in a very easy, simple way. However, there are several problems with this strategy. First, it requires learning many different languages, typically with very magic syntaxes packed with random symbols that do various things, with no clear connection between the symbols and the actions. Second of all, the simplified formats never end up covering all use cases, resulting in hacky workarounds to achieve the desired functionality, or, in some cases, entirely new languages that promise to cover all of the use cases, for real this time.

The eventual result of this trend is that people end up stuffing entire programming languages into these supposedly simple formats to cover their requirements (for example, SCSS, JSX, and templates). The resulting languages are neither simple, clear, nor extensible, creating massive amounts of unreadable and inefficient code. This means that everyone has to learn these languages and their magic syntaxes, struggle to read their complicated code, reach their inevitable limits in functionality, and suffer runtime errors and bad performance due to the typically duck-typed, interpreted nature of these languages.

The solution to this is simple: whenever possible, everything should be written in real code, preferably in one language. Therefore, Cogent Core takes this approach: everything, from [[doc:tree]]s to [[widget]]s to [[style]]s to [[enum]]s, is written in real, pure Go code. The only non-Go functional files in a Cogent Core package or app should be TOML files, which are only used for very simple configuration options to commands, and not for any actual code.

### Closures everywhere

An important implication of the above is that _closure functions_ should be used anywhere that any kind of configuration is needed, instead of trying to parameterize the space of options that might likely be needed. A closure can contain arbitrary code-based logic while capturing arbitrary external context needed to conditionalize this logic. Thus, closure functions are used everywhere in Cogent Core, including [[style]]r functions, Maker functions in [[plan]]s, [[event]] handlers, etc.

## Go is the best programming language (for GUIs)

There are many programming languages, and each one represents a different set of tradeoffs. We think that Go is the best language (for GUIs and most other use-cases) according to the following list of fundamental characteristics:

### Type safety and compilation

One of the most important features of a programming language is type safety. Type safety and compilation give helpful compile-time errors, code completion, and syntax highlighting that make code safe and readable. This requirement eliminates many popular languages like JavaScript and Python, where it is common for simple bugs to lurk unnoticed until each line of code is executed at runtime.

### Simplicity and elegance

Code must be *simple* to write. Humans must read and write code, and it is much easier to read and write code with simple, semantic, and consistent operations, not symbol soup like that found in Regex, SCSS, and Rust macros (what does `($($($([` do?). Humans should not have to waste time writing semicolons and parentheses around their `if` conditions. Humans should not have to redundantly repeat the type of something (`Something something = Something{};`). Type safety should make code safe, not intricate and verbose. This condition eliminates many compiled languages like Rust and C++ (note: Rust and C++ are useful for certain things like operating system kernels and graphics drivers, but this is about end-user GUI development).

### Speed

Programs must be quick to write, compile, and run. Most compiled languages fail the first and second conditions, and most interpreted languages fail the last. Go can be written extremely quickly due to its simple and elegant syntax, it compiles in seconds even for complex GUIs, and there is no runtime performance deficit significant enough to impact GUIs.

### Cross-platform and cross-compilation

Every widely used platform should be well supported, and to the greatest extent possible, the exact same code should work as expected on every platform. Furthermore, because Go's compiler is written in Go, it is relatively straightforward to cross-compile for other platforms. By contrast, JavaScript has to deal with the notorious inconsistency across different browsers, and because most of Python requires further C / C++ code to be compiled (to manage the performance-critical aspects), it is often a nightmare to manage all the dependencies and environment variables required to get something to work on a given platform, at the right version, etc.

### Robust versioning and package management

Go's `module` based versioning and package management system is built into the language, and is extremely robust and simple to use. By contrast, dependencies and package management in JavaScript and Python is notoriously bad.

### Rationality and consistency

Languages must be rational and consistent both internally and with other programming languages. If other programming languages use `object.Method(args)` to call a method, and you use `function(args)` to call a function, then maybe you should not use `[object method:args]` to call methods (Objective-C). `SCREAMING_SNAKE_CASE` may help you vent frustration at the terrible programming language you are using, but it is not good for clean and readable code. If `10+"1"` is `"101"`, then maybe `10-"1"` shouldn't be `9` (JavaScript). Two of the most core operations in a programming language, `var` and `==`, should not be semantically incorrect and require the use of alternative operators instead, `let` and `===` (also JavaScript).

## Go language-specific considerations

### Struct fields are better than maps for things like configuration and styling

Configuration settings, typically settable with a config file and / or command-line arguments, are stored as key-value maps in the widely used [cobra](https://github.com/spf13/cobra), viper and other such tools. Likewise, in early development of Cogent Core, styling was set using Properties maps.

However, using a `struct` with appropriately named fields has the following advantages:
* Compile time name-safety: the compiler ensures you didn't mistype the key name.
* Compile time type-safety: the type of the property is not `any` but the actual type needed.
* Tab completion and full lookup in IDEs -- much easier when _using_ config values in an app, and also when setting styling in GUI.
* GUI editor of config opts as a [[form]] has full access to field tag GUI hints, etc.
* [[generate#types]] can provide access to field comments for full docs for each option, whereas the map implementation requires separate maps of docs and values.

This is why the [[cli]] configuration and app command management system is based on structs, and why Cogent Core uses direct styling functions that directly set values on the [[doc:styles.Style]] structs.

### Generate vs. `reflect` tradeoffs

[[Generate]]d code is, in general, faster and cleaner and can be targeted to just what is needed. On the other hand, `reflect` is more robust than generated code (doesn't require proper such code to have been generated), and generated code takes more space.  Therefore, these tradeoffs need to be considered in a case-by-case basis, also in conjunction with generics.

* [[generate#types]] went back-and-forth from using generated vs reflect code, and settled on an "optimal" hybrid of the two. The documentation-level data extracted from comments can only be obtained using generate, but we saved a lot of code by using reflect + generics instead of generating a `Type` variable for each type.

* Some [[widget]]s use `reflect` to operate on any Go type, such as [[form]] on `struct` and [[list]] on slices.

### Interfaces instead of `reflect`

Interfaces go hand-in-hand with generated code: the boilerplate code that satisfies the interfaces is auto-generated as needed.

The prototype here is [[enum]]s.

### Keep interfaces small

This is a Go mantra that can be difficult for people coming from other languages such as C++, where there is a temptation to specify all functionality in a header-file like interface, and then implement that with a base type.

The general principle is simple: _an interface should only specify the methods that actually need to be implemented differently_.

For example, the [[doc:tree.Node]] interface used to be rather large, but now we follow the above principle and just require people to do `AsTree()` to access all the "base" level functionality defined on [[doc:tree.NodeBase]].  This also helps to avoid naming conflicts between the interface methods and the implementation fields.

### Exporting vs. hiding

One extreme strategy is to hide (lowercase) everything and only export (uppercase) the minimal set of functionality, exclusively through methods, including `Set` and `Get` accessors for all fields.  While this is maximally "future proof" and robust (e.g., when `Set` might have side-effects or not), it results in significant boilerplate code to provide all the accessors.

A more relaxed strategy, used frequently in the Go stdlib, is to export `struct` fields directly wherever possible, and only provide `Set` methods where side-effects exist. Furthermore, such side-effects should be minimized, including handling the "zero value" case to the extent possible, such that no initializer methods are needed.  Making fields exported also allows direct inspection of such fields in the GUI inspector, providing important easy-to-use debugging and understanding.

This is the approach taken in Cogent Core (along with auto-generated `Set` methods for chaining as discussed below).  In general, setting fields should have no side-effects, and instead all the side-effects should be handled in the [[doc:core.WidgetBase.Update]] method (via [[update]]rs or [[style]]rs as appropriate).

In C++ and other languages, it is of utmost importance to hide all implementational details, to ensure that minor changes do not affect the binary linking compatibility of dynamic libraries. However, in Go, everything is always built from source and no such consideration really applies.  Therefore, it is possible to be more relaxed about hiding things.  Furthermore, it is all-too-often the case that someone has hidden some key bit of implementation that is critical for some other package to use, resulting in significant painful code duplication.

Nevertheless, hiding implementational details makes it easier for users to find the relevant bits of the API (and reduces the size of the pkg.go.dev docs).  Furthermore, _it is always possible to export something that was hidden, without negative consequences for API compatibility_, whereas once something is exported, someone might end up using it, and that might cause some issues down the road if it is changed.

Therefore, _if anyone ever needs access to something that has been hidden, we are more than happy to export it!_ -- just file an issue.

### Chaining function calls

The [[generate#types]] package performs automatic generation of `New` and `Set` convenience methods that have one key advantage over direct field access: they return a pointer to the receiver object, so that you can chain multiple calls together, as in:

```Go
core.NewButton(b).SetText("Send").SetIcon(icons.Send).OnClick(func(e events.Event) {
	core.MessageSnackbar(b, "Message sent")
})
```

This can save considerable vertical space, and avoids any temptation to bundle multiple field setters into one function with multiple arguments, which then become more ambiguous.

## Code organization

### Packages should be small, focused, and minimal

We find the coherent chunks of functionality and encapsulate them in separate packages, instead of creating sprawling mega-packages like `ki` was before.  Now that we know all the functionality we need, we can think more carefully about how to package it.

Try to make these packages as independent as possible; don't have them depend on other infrastructure unless absolutely necessary, so that they can be used by anyone in an unencumbered way.

This is what we look for in considering whether to import a given package, so it is what we should provide.

Examples:
* [[doc:colors]] pulled out of core
* [[doc:reflectx]] pulled reflection stuff out of tree
* [[doc:clicore]] separated from [[doc:cli]] to keep cli free of core dependency

### Use function libraries instead of putting lots of methods on a type

Go uses the `strings` package instead of adding a lot of builtin methods on the `string` type. The advantages are:

* More flexibility to add and change implementations; can even have a replacement implementation with the same signatures.
* Multiple different such packages can be used for different subsets of functionality (e.g., regex, unicode etc are all separate).
* Type itself remains small and only handles most basic functionality; presumably this works better for checking interface implementation etc.

Consistent with this approach, [[doc:colors]] implements functions operating on the standard `color.RGBA` data type, instead of the many methods we defined on `gist.Color` in the old version.

## Naming

In general, we follow the Go and Google standards:

* https://go.dev/doc/effective_go#names
* https://go.dev/wiki/CodeReviewComments
* https://google.github.io/styleguide/go/decisions

Consistent with the minimalist Go aesthetic, wherever possible, we try to find the shortest _full_ name for any element, that captures its meaning well.  Although abbreviations are widely used and ever so tempting to save space and remove code clutter, it is better if a complete and _short_ name can be found instead.

Furthermore, additional effort should be spent to avoid generic names such as `Manager`, and adding other generic terms like `Item`, `Data`, `State`, `Run`, `Compute`, etc to names.  You'll find that they are usually superfluous, and almost everything a computer does involves these things, so just focus on what uniquely applies to the particular case you're trying to name.

Here is a list of choices for cases where there are two widely-used but equivalent names: use only one of them everywhere for consistency:

* `Delete` instead of `Remove`

### Collections are plural

Packages and types whose primary purpose is to describe/hold a collection of something should be named in the plural. For example, [[doc:events]] holds a collection of different event types, so it has a plural name. Also, per this rule, enum types are almost always named in the plural, as they describe a collection of possible values.
