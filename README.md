# ki
Part of the GoKi Go language (golang) full strength tree structure system (ki = tree in Japanese)

`package ki` -- core `Ki` interface (`ki.go`) and `Node` struct (`node.go`), plus other supporting players.

[![Go Report Card](https://goreportcard.com/badge/github.com/rcoreilly/goki/ki)](https://goreportcard.com/report/github.com/rcoreilly/goki/ki)
[![GoDoc](https://godoc.org/github.com/rcoreilly/goki/ki?status.svg)](http://godoc.org/github.com/rcoreilly/goki/ki)

A Ki tree is recursively composed of Ki Node structs, in a one-Parent / multiple-Child structure.  The typical use is to embed Node in other structs that then implement specific tree-based functionality.  See other packages in GoKi for examples.

# Code Map

* `kit` package: `kit.Type` struct of `reflect.Type` that supports saving / loading of type information using `kit.Types` `TypeRegistry` -- provides name to type map for looking up types by name, and types can have default properties. `kit.Enums` `EnumRegistry` provides enum (const int) <-> string conversion, including `bitflag` enums.  Also has robust generic `ki.ToInt` `ki.ToFloat` etc converters from `interface{}` to specific type, for processing properties

* `bitflag` package: simple bit flag setting, checking, and clearing methods that take bit position args as ints (from const int eunum iota's) and do the bit shifting from there

* `ki.go` = `Ki` interface for all major tree node functionality

* `ptr.go` = `ki.Ptr` struct that supports saving / loading of pointers using paths

* `slice.go` = `ki.Slice []Ki` supports saving / loading of Ki objects in a slice, by recording the size and types of elements in the slice -- requires `ki.Types` type registry to lookup types by name

* `signal.go` = `Signal` that can call any number of functions with receiver Ki objects that have been previously `Connect`ed to the signal -- also supports signal type so the same signal sender can send different types of signals over the same connection -- used for signaling changes in tree structure, and more general tree updating signals.

# Go Language (golang) Notes (esp for people coming from C++)

general summary: https://github.com/golang/go/wiki/GoForCPPProgrammers

## Naming Conventions

https://golang.org/doc/effective_go.html#names

* Don't use Get* -- just use the name itself, when a "getter" is necessary

* Do use Set* -- setter + field name -- this is convention in Qt too

* Uppercase all Fields in general -- Strongly prefer to just deal with the fields directly without having to go through a getter method, and we want embedded objects and anyone else to be able to access those fields, and they also need to be saved / loaded through JSON, etc

* Above means that there can be conflicts for any interfaces that need to provide getter access to those fields.  bummer.  Using Ki prefix in those cases.

* `Delete` instead of `Remove`

* Unless it is a one-liner converter to a given type or value like Stringer, it can be challenging to name an interface and a base type for that interface differently.
	+ The Interface should generally be given priority, and have the cleaner name.  Base types are only typed relatively rarely at start of structs that embed them, so they are less important.
	+ One not-so-good idea: Add a capital I at the end of an interface when it is designed for derived types of a given base, e.g., `EventI` for structs that embed type `Event` -- I couldn't find anything about this in searching but somehow it doesn't seem like the "Go" way..
	
* It *IS* ok to have types and fields / members of the same name!  So EventType() EventType is perfectly valid and that's a relief :)

* It is hard to remember, but important, that everything will be prefixed by the package name for users of the package, so *don't put a redundant prefix on anything*

* Use `AsType()` for methods that give you that give you that concrete type from a struct (where it isn't a conversion, just selecting that view of it)

### Enums (const int)

* Use plural for enum type, instead of using a "Type" suffix -- e.g., `NodeSignals` instead of `NodeSignalType`, and in general use a consistent prefix for all enum values: NodeSignalAdded etc 

* my version of stringer generates conversions back from string to given type

* ki.EnumRegister (ki.AddEnum) -- see types.go -- adds a lot of important functionality to enums

## Struct structure

* ALL Nodes need to be in Children of parent -- not e.g., as fields in a struct (unless they are definitely starting a new root at that point, in which case it is fine).  Use an InitNode kind of constructor to build default children that are then equivalent to fields -- always there, accessible by name, etc.

## Interfaces, Embedded types

* In C++ terms, an interface in go creates a virtual table -- anytime you need virtual functions, you must create an interface.
	+ WARNING: you need to define the *entire* interface for *each* type that implements it -- there is *no inheritance* in Go!  Thus, it is important to keep interfaces small!  Or, in the case of `Ki` / `Node`, only have one struct that implements it.
	+ An interface is the *only* way to create an equivalence class of types -- otherwise Go is strict about dealing with each struct in terms of the actual type it is, *even if it embeds a common base type*.
	
* Anonymous embedding a type at the start of a struct gives transparent access to the embedded types (and so on for the embedded types of that type), so it *looks* like inheritance in C++, but critically those inherited methods are *not* virtual in any way, and you must explicitly convert a given "derived" type into its base type -- you cannot do something like: `bt := derived.(*BaseType)` to get the base type from a derived object -- it will just complain that derived is its actual type, not a base type.  Although this seems like a really easy thing to fix in Go that would support derived types, it would require an expensive dynamic (runtime) traversal of the reflect type info.  Those methods are avail in package `kit` as `TypeEmbeds` to check if one type embeds another, and `EmbededStruct` to get an embedded struct out of a struct that embeds it (at any level of depth).  In general it is much better to provide an explicit interface method to provide access to embedded structs that serve as base-types for a given level of functionality.  Or provide access to everything you might need via the interface, but just giving the base struct is typically much easier.

* Although the derived types have direct access to members and methods of embedded types, at any level of depth of embedding, the `reflect` system presents each embedded type as a *single field* as it is declared in the actual code.  Thus, you have to recursively traverse these embedded structs -- methods to flatten these field lists out by calling a given function on each field in turn are provided in `kit embed.go`: `FlatFieldsTypeFun` and `FlatFieldsValueFun`

## Closures & anonymous functions

It is very convenient to use anonymous functions directly in the `FunDown` (etc) and `Signal Connect` cases, but for performance reasons, it is important to be careful about capturing local variables from the parent function, thereby creating a *closure*, which creates a local stack to represent those variables.  In the case of FunDown / FunUp etc, the impact is minimized because the function is ONLY used during the lifetime of the outer function.  However, for `Signal Connect`, the function is itself saved and used later, so using a closure there creates extra memory overhead for each time the connection is created.  Thus, it is generally better to avoid capturing local variables in such functions -- typically all the relevant info can be made available in the recv, send, sig, and data args for the connection function.

## Notes on things that would perhaps be nice to change about Go..

As the docs state, you really have to use it for a while to appreciate all of the design decisions in the language, so these are very preliminary and subject to later reconsideration.. 

* Reference arg type -- the receiver arg (the "this" pointer in C++) has
  automagical ability to either be a pointer or a value depending on the
  function declaration, but other args do NOT have this flexibility.  It means
  that the caller has to do a little bit of extra work adding a & or not where
  necessary.  The counter-argument is presumably that this means that the
  caller KNOWS that by passing a pointer, the value of the arg will be
  changed, and that is probably a good thing.  The same point could be made
  for the receiver arg, but there I suppose the syntax would have been a bit
  clunky and confusing.

# TODO

* XML IO -- first pass done, but more should be in attr instead of full elements
* FindChildRecursive functions?
* port to better logging for buried errors, with debug mode: https://github.com/sirupsen/logrus
