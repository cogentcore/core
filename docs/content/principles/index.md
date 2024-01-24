# Principles

This section contains a list of fundamental principles that Cogent Core is built on. There are occasional exceptions to these principles, but they should be followed almost all of the time, and there must be a very clear reason for not following them.

# Code is always the best solution

A common approach to programming, especially for the web, is to reduce the amount of actual programming that takes place and instead use simplified markup, configuration, and selection formats like CSS, HTML, YAML, and Regex. The promise of this is that you will be able to accomplish everything you need to do in a very easy, simple way. However, there are several problems with this strategy. Firstly, it requires learning many different languages, typically with very magic syntaxes packed with random symbols that do various things, with no clear connection between the symbols and the actions. Second of all, the simplified formats never end up covering all use cases, resulting in hacky workarounds to achieve the desired functionality, or, in some cases, entirely new languages that promise to cover all of the use cases, for real this time.

The eventual result of this trend is that people end up stuffing entire programming languages into these supposedly simple formats to cover their requirements (for example, SCSS, JSX, and templates). The resulting languages are neither simple, clear, nor extensible, creating massive amounts of unreadable and inefficient code. This means that everyone has to learn these languages and their magic syntaxes, struggle to read their complicated code, reach constant limits in functionality, and suffer runtime errors and bad performance due to the typically duck-typed, interpreted nature of these languages. Unless there are very limited uses for something, avoiding real code will always cause many problems and no benefits. 

The solution to this is simple—whenever possible, everything should be written in real code, preferably in one language. Therefore, Goki takes this approach: everything, from trees to widgets to styling to enums, is written in real, pure Go code. The only non-Go functional files in a Goki package or app should be TOML files, which are only used for very simple configuration options to commands, and not for any actual code.

# Struct fields are better than maps for things like configuration and styling

Configuration settings, typically settable with a config file and / or command-line arguments, are stored as key-value maps in the widely used [cobra](https://github.com/spf13/cobra), viper and other such tools.  Likewise, in v1 of GoGi, styling was set using Props maps.

However, using a `struct` with appropriately-named fields has the following advantages:
* Compile time name-safety: the compiler ensures you didn't mistype the key name.
* Compile time type-safety: the type of the property is not `any` but the actual type needed.
* Tab completion and full lookup in IDEs -- much easier when _using_ config values in an app, and also when setting styling in GUI.
* GUI editor of config opts as a StructView has full access to field tag GUI hints, etc.
* [gti](https://github.com/goki/gti) can provide access to field comments for full docs for each option -- the map impl requires  separate maps of docs vs. values.

This is why the [grease](https://github.com/goki/grease) configuration and app command management system is based structs, and v2 of GoGi uses "direct styling" functions that directly set values on the `styles.Style` style structs.

# Generate instead of `reflect`

Generated code is faster and cleaner and can be targeted to just what is needed.  `reflect` should be reserved for things like `giv.StructView` and other such views which need to be truly generic and operate on any kind of type from any package, etc.

# Interfaces instead of `reflect`

Interfaces go hand-in-hand with generated code: the boilerplate code that satisfies the interfaces is auto-generated as needed.

The prototype here is [enums](https://github.com/goki/enums)

# Repositories and packages should be small, focused, and minimal

Find the coherent chunks of functionality and encapsulate them in separate repositories, instead of creating sprawling mega-repos like `ki` was before.  Now that we know all the functionality we need, we can think more carefully about how to package it.

Try to make these repos as independent as possible -- don't have them depend on other goki infrastructure unless absolutely necessary, so that they can be used by anyone in an unencumbered way.

This is what we look for, in considering whether to import a given package, so it is what we should provide.

Examples:
* [colors](https://github.com/goki/colors) -- pulled out of gi
* [laser](https://github.com/goki/laser) -- pulled reflection stuff out of kit
* [greasi](https://github.com/goki/greasi) separated from [grease](https://github.com/goki/grease) to keep grease free of gi dependency.

# Use function libraries instead of putting lots of methods on a type

Go uses the `strings` package instead of adding a lot of builtin methods on the `string` type.  The advantages are:
* More flexibility to add and change impls -- can even have a replacement impl with the same signatures.
* Multiple different such packages can be used for different subsets of functionality (e.g., regex, unicode etc are all separate).
* Type itself remains small and only handles most basic functionality -- presumably this works better for checking Interface implementation etc.

Consistent with this approach, [colors](https://github.com/goki/colors) implements functions operating on the standard `color.RGBA` data type, instead of the many methods we defined on `gist.Color` in V1.
