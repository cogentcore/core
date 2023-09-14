# enums

Enums provides utilities for creating and using enumerated types in Go. There are two main parts of enums: package `enums`, which defines the `Enum` and `BitFlag` interfaces, and the enum code generation tool `enumgen`, which is based on [dmarkham/enumer](https://github.com/dmarkham/enumer) and [stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer).

Add one of the two following comment directives after the `type` declaration of an enum:
* `//enums:enum` -- for standard enum
* `//enums:bitflag` -- for bit flag enum (see below for details)


# BitFlag enums

This package implements bit flag enums using enum values that specify a _bit index_ for the flag, which is then used with bit shifting to create the actual bit mask.  Thus, the enum values are just sequential integers like a normal enum, which allows the names to be looked up using a slice index, and are generally easier to read and understand as simple integers.

The generated `String()` and `SetString()` methods operate using the bit-shifted mask values and return the set of active bit names (separated by an OR pipe `|`) for a given value.  Use `BitIndexString()` to get the name associated with the bit index values (which are typically only used for setting and checking flags).


