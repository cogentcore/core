# Enums in Go

Enums provides utilities for creating and using enumerated types in Go. There are two main parts of enums: command `enumgen` and package `enums`. Most end-users should only have to interact with `enumgen`.

## Enumgen

Enumgen generates code for enumerated types that aids in various operations with them, including printing, setting, and marshalling. Enumgen is based on [dmarkham/enumer](https://github.com/dmarkham/enumer) and [stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer). To install enumgen, run:

```sh
go install cogentcore.org/core/enums/cmd/enumgen@latest
```

Then, in any package you have enums, add *one* `go:generate` line at the top of a file (**only** one per package):

```go
package mypackage

//go:generate enumgen

...
```

Enum types are simply defined as any other Go type would be, just with a comment directive after it. Standard enum types can be defined as any signed or unsigned integer type, although `int32` is preferred because enum types could need to be big but never need to be giant. Bit flag enum types must be `int64`; see [Bit flag enums](#bit-flag-enums) for why.

After the type declaration of *each* enum, add one of the following two comment directives:

* `//enums:enum` for standard enums
* `//enums:bitflag` for bit flag enums (see [Bit flag enums](#bit-flag-enums) for more information)

Then, declare your enum values in a constant block using `iota`. The constant values can be unsequential, offset, and/or negative (although bit flags can not be negative), but most enums are typically none of those. Also, you can declare aliases for values. For example:

```go
package mypackage

//go:generate enumgen

type MyEnum int32 //enums:enum

const (
    Apple MyEnum = iota
    Orange
    Peach
)

type MyBitFlagEnum int64 //enums:bitflag

const (
    Enabled MyBitFlagEnum = iota
    Disabled
    Focused
    Hovered
)

type MyComplicatedEnum int16 //enums:enum

const (
    Go MyComplicatedEnum = -2 * iota + 1
    Python
    ObjectiveC
    JavaScript
    WorstProgrammingLanguage = ObjectiveC
    // alias ^
)
```

Any time you add, remove, or update enums, run `go generate`.

The behavior of enumgen can be customized in various ways through flags on *either* the package-level `go:generate` line or the enum-specific comment directive. Run `enumgen -h` in your terminal to learn about these flags. Here is a simple example of flag setting:

```go
package mypackage

//go:generate enumgen -no-text -transform snake

type MyEnum uint32 //enums:enum -add-prefix fruit_ -no-line-comment -sql

const (
    Apple MyEnum = iota
    Orange
    Peach
)
```

## Package enums

Package enums defines standard interfaces that enums satisfy.

* `Enum` is satisfied by all enum types
* `EnumSetter` is satisfied by all pointers to enum types
* `BitFlag` is satisfied by all bit flag enum types
* `BitFlagSetter` is satisfied by all pointers to bit flag enums types

## Documenting enums

Enumgen captures any *doc* comments (**not** line comments) you place on enum values and exposes them through the `Desc` method. For example, if you have:

```go
package mypackage

//go:generate enumgen

type Days uint8 //enums:enum

const (
	// Sunday is the first day of the week
	Sunday Days = iota
	// Monday is the second day of the week
	Monday
	// Tuesday is the third day of the week
	Tuesday
)
```

Then `Sunday.Desc()` will be `Sunday is the first day of the week`.

## Bit flag enums

Bit flag enums are just enums that are **not** mutually exclusive, so you can have multiple of them specified at once. Each option/flag that can be specified occupies one bit, meaning that you should have a large type to avoid running out of space. Therefore, enumgen currently requires all bit flags to be of type `int64`, and there can be no more than 64 values for a bit flag enum type.

This package implements bit flag enums using enum values that specify a _bit index_ for the flag, which is then used with bit shifting to create the actual bit mask.  Thus, the enum values are just sequential integers like a normal enum, which allows the names to be looked up using a slice index, and are generally easier to read and understand as simple integers.

The generated `String()` and `SetString()` methods operate using the bit-shifted mask values and return the set of active bit names (separated by an OR pipe `|`) for a given value.  Use `BitIndexString()` to get the name associated with the bit index values (which are typically only used for setting and checking flags).

## Extending enums

You can define an enum as extending another enum, which allows you to inherit all of its values and then build on top of them. To do so, you must define the type of the new enum as an extension of the type of the enum you are extending. Furthermore, you must define the first value of the new enum as the `N` value of the enum you are extending. For example:

```go
package mypackage

//go:generate enumgen

type Fruits int //enums:enum

const (
	Apple Fruits = iota
	Orange
	Peach
)

type Foods Fruits //enums:enum

const (
	Bread Foods = Foods(FruitsN) + iota
	Lettuce
	Cheese
)
```

**NOTE:** the `N` value is generated by enumgen, so you need to run `go generate` with the enum you are extending already declared *before* you declare your new enum. If you screw up this ordering and get an error or a panic when running `go generate`, you can temporarily comment out the new enum, run `go:generate`, and then add it back again and run `go:generate` again.

**NOTE:** in the rare case that you have an enum type that extends a non-enum, non-builtin type (eg: `fixed.Int26_6`), you need to specify the `-no-extend` flag in your `//enums:enum` comment directive to prevent errors. For example:

```go
type MyEnum fixed.Int26_6 //enums:enum -no-extend
```
