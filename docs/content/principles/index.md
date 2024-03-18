# Principles

This section contains a list of fundamental principles that Cogent Core is built on. There are occasional exceptions to these principles, but they should be followed almost all of the time, and there must be a very clear reason for not following them.

# Code is always the best solution

A common approach to programming, especially for the web, is to reduce the amount of actual programming that takes place and instead use simplified markup, configuration, and selection formats like CSS, HTML, YAML, and Regex. The promise of this is that you will be able to accomplish everything you need to do in a very easy, simple way. However, there are several problems with this strategy. Firstly, it requires learning many different languages, typically with very magic syntaxes packed with random symbols that do various things, with no clear connection between the symbols and the actions. Second of all, the simplified formats never end up covering all use cases, resulting in hacky workarounds to achieve the desired functionality, or, in some cases, entirely new languages that promise to cover all of the use cases, for real this time.

The eventual result of this trend is that people end up stuffing entire programming languages into these supposedly simple formats to cover their requirements (for example, SCSS, JSX, and templates). The resulting languages are neither simple, clear, nor extensible, creating massive amounts of unreadable and inefficient code. This means that everyone has to learn these languages and their magic syntaxes, struggle to read their complicated code, reach constant limits in functionality, and suffer runtime errors and bad performance due to the typically duck-typed, interpreted nature of these languages. Unless there are very limited uses for something, avoiding real code will always cause many problems and no benefits. 

The solution to this is simple—whenever possible, everything should be written in real code, preferably in one language. Therefore, Cogent Core takes this approach: everything, from trees to widgets to styling to enums, is written in real, pure Go code. The only non-Go functional files in a Cogent Core package or app should be TOML files, which are only used for very simple configuration options to commands, and not for any actual code.

# Go is the only good programming language (for GUIs)

There are many programming languages. However, almost all of them lack at least one of several fundamental characteristics required to make a good programming language for GUIs, making Go the only good programming language, and thus the basis for the Cogent Core framework.

### Type safety and compilation

One of the most important features of a programming language is type safety. Type safety and compilation give helpful compile-time errors, code completion, and syntax highlighting that make code safe and readable. This requirement eliminates many popular languages like JavaScript and Python.

### Simplicity and elegance

Code must be *simple* to write. Humans must read and write code, and it is much easier to read and write code with simple, semantic, and consistent operations, not symbol soup like that found in Regex, SCSS, and Rust macros (what does `($($($([` do?). Humans should not have to waste time writing semicolons and parentheses around their `if` conditions. Humans should not have to repeat the type of something a million times (`Something something = Something{};`). Type safety should make code safe, not intricate and verbose. This condition eliminates many compiled languages like Rust and C++ (note: Rust and C++ are useful for certain things like operating system kernels and graphics drivers, but this is about end-user GUI development).

### Speed

Programs must be quick to write, compile, and run. Most compiled languages fail the first and second conditions, and most interpreted languages fail the last. Go can be written extremely quickly due to its simple and elegant syntax, it compiles in seconds even for complex GUIs, and there is no runtime performance deficit significant enough to impact GUIs.

### Cross compilation



### Rationality and consistency

Languages must be rational and consistent both internally and with other programming languages. If other programming languages use `object.Method(args)` to call a method, and you use `function(args)` to call a function, then maybe you should not use `[object method:args]` to call methods (Objective-C). `SCREAMING_SNAKE_CASE` may help you vent frustration at the terrible programming language you are using, but it is not good for clean and readable code. If `10+"1"` is `"101"`, then maybe `10-"1"` shouldn't be `9` (JavaScript). Two of the most core operations in a programming language, `var` and `==`, should not be semantically incorrect and require the use of alternative operators instead, `let` and `===` (also JavaScript).

# Struct fields are better than maps for things like configuration and styling

Configuration settings, typically settable with a config file and / or command-line arguments, are stored as key-value maps in the widely used [cobra](https://github.com/spf13/cobra), viper and other such tools.  Likewise, in v1 of GoGi, styling was set using Props maps.

However, using a `struct` with appropriately-named fields has the following advantages:
* Compile time name-safety: the compiler ensures you didn't mistype the key name.
* Compile time type-safety: the type of the property is not `any` but the actual type needed.
* Tab completion and full lookup in IDEs -- much easier when _using_ config values in an app, and also when setting styling in GUI.
* GUI editor of config opts as a StructView has full access to field tag GUI hints, etc.
* [[gti]] can provide access to field comments for full docs for each option -- the map impl requires  separate maps of docs vs. values.

This is why the [[grease]] configuration and app command management system is based structs, and v2 of GoGi uses "direct styling" functions that directly set values on the `styles.Style` style structs.

# Generate instead of `reflect`

Generated code is faster and cleaner and can be targeted to just what is needed.  `reflect` should be reserved for things like `giv.StructView` and other such views which need to be truly generic and operate on any kind of type from any package, etc.

# Interfaces instead of `reflect`

Interfaces go hand-in-hand with generated code: the boilerplate code that satisfies the interfaces is auto-generated as needed.

The prototype here is [[enums]]

# Packages should be small, focused, and minimal

Find the coherent chunks of functionality and encapsulate them in separate packages, instead of creating sprawling mega-packages like `ki` was before.  Now that we know all the functionality we need, we can think more carefully about how to package it.

Try to make these packages as independent as possible; don't have them depend on other infrastructure unless absolutely necessary, so that they can be used by anyone in an unencumbered way.

This is what we look for, in considering whether to import a given package, so it is what we should provide.

Examples:
* [[colors]] pulled out of gi
* [[laser]] pulled reflection stuff out of kit
* [[greasi]] separated from [[grease]] to keep grease free of gi dependency

# Use function libraries instead of putting lots of methods on a type

Go uses the `strings` package instead of adding a lot of builtin methods on the `string` type.  The advantages are:
* More flexibility to add and change impls -- can even have a replacement impl with the same signatures.
* Multiple different such packages can be used for different subsets of functionality (e.g., regex, unicode etc are all separate).
* Type itself remains small and only handles most basic functionality -- presumably this works better for checking Interface implementation etc.

Consistent with this approach, [[colors]] implements functions operating on the standard `color.RGBA` data type, instead of the many methods we defined on `gist.Color` in V1.
