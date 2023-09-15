# glop

Package `glop` is Go Language Outtake Packages: Stuff that should have been in the Go standard library, and convenient wrappers that make high-frequency steps more convenient (often using generics).

Everything in here is very short and lightweight -- anything requiring more than a few lines of code should be in its own separate repository.

* [dirs](dirs) has various file-system directory-related functions, including a simple `FileExists` and various convenience versions for getting a slice of files in a directory.

* [num](num) has a generic version of `Abs`, a duplicate set of numeric constraints from the std `constraints` package with the much shorter package name of `num` and including the overlooked `Number`, a generic way to set any number to any other number, and conversion between numbers and bools.

* [bools](bools) for a `Booler` and `BoolSetter` interface, and To/From String methods.

* [indent](indent) provides space and tab based indent string generators.

* [runes](runes) provides some key elements of `strings` and `bytes` operating on a `[]rune` slice, for faster conversion-free performance in cases where you have a slice of runes.

* [atomctr](atomctr) provides convenient method wrappers around `sync/atomic` for the common case of a shared `int64` counter.

* [fatomic](fatomic) provides wrappers around `sync/atomic` methods for doing atomic operations on floating point numbers.

