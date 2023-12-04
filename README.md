# enums

Enums provides utilities for creating and using enumerated types in Go. There are two main parts of enums: command `enumgen` and package `enums`. Most end-users should only have to interact with `enumgen`.

## Enumgen

Enumgen generates code for enumerated types that aids in various operations with them, including printing, setting, and marshalling. Enumgen is based on [dmarkham/enumer](https://github.com/dmarkham/enumer) and [stringer](https://pkg.go.dev/golang.org/x/tools/cmd/stringer). To install enumgen, run:

```sh
go install goki.dev/enums/cmd/enumgen@latest
```

Then, in any package you have enums, add *one* `go:generate` line at the top of a file (**only** one per package):

```go
package mypackage

//go:generate enumgen

...
```

After the type declaration of *each* enum, add one of the following two comment directives:

* `//enums:enum` for standard enums
* `//enums:bitflag` for bit flag enums (see below for details)

For example,

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
    Enabled MyBitFlagEnum
    Disabled
    Focused
    Hovered
)
```

Any time you add, remove, or update enums, run `go generate`. The behavior of enumgen can be customized in various ways through flags on *either* the package-level `go:generate` line or the enum-specific comment directive. Run `enumgen -h` in your terminal to learn about these flags. Here is a simple example of flag setting:

```go
package mypackage

//go:generate enumgen -json -transform snake

type MyEnum int32 //enums:enum -add-prefix fruit_ -no-line-comment -sql

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


## Bit flag enums

Bit flag enums are just enums that are **not** mutually exclusive, so you can have multiple of them specified at once. Each option/flag that can be specified occupies one bit, meaning that you should have a large type to avoid running out of space. Therefore, enumgen currently requires all bit flags to be of type `int64`, and there can be no more than 64 values for a bit flag enum type.

This package implements bit flag enums using enum values that specify a _bit index_ for the flag, which is then used with bit shifting to create the actual bit mask.  Thus, the enum values are just sequential integers like a normal enum, which allows the names to be looked up using a slice index, and are generally easier to read and understand as simple integers.

The generated `String()` and `SetString()` methods operate using the bit-shifted mask values and return the set of active bit names (separated by an OR pipe `|`) for a given value.  Use `BitIndexString()` to get the name associated with the bit index values (which are typically only used for setting and checking flags).


