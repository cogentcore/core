# laser: reflection optimized

[![Go Report Card](https://goreportcard.com/badge/goki.dev/laser)](https://goreportcard.com/report/goki.dev/laser)
    <a href="https://pkg.go.dev/goki.dev/laser"><img src="https://img.shields.io/badge/dev-reference-007d9c?logo=go&logoColor=white&style=flat" alt="pkg.go.dev docs"></a>

Package laser is a highly reflective package of golang reflect helpers (lasers work by bouncing light back and forth between two mirrors).  It also connotes the "lazy" aspect of reflection -- running dynamically instead of statically precompiled.  It is derived from the [ki/kit](https://github.com/goki/ki/tree/v1) package in v1 of Ki.

As usual, Go [reflect](https://pkg.go.dev/reflect) provides just the minimal API for dealing with reflection, and there are several well-documented issues that require a bit of non-obvious logic to get around.

Some example functions:

* `AnyIsNil()` -- checks if interface value is `nil` -- requires extra logic for multiple levels of nil.

* `ValueIsZero()` -- checks for any kind of zero, including `nil`.

* `ToInt, ToFloat, ToString, ToBool` -- robustly converts any value to given type, using an efficient type switch for all basic types, including pointers, and using `strconv` parse from string.  See also [glop/num](https://github.com/goki/glop/tree/num) for generics-based conversion.  The key difference is that if you have an `any`, you can't use generics, so these type-switch methods are necessary in that case.

* `SetRobust(to, frm any) bool` -- robustly sets the 'to' value from the 'from' value, for any case, using the To* functions and JSON format for maps and slices.

* `UnhideAnyValue(v reflect.Value) reflect.Value` -- ensures value is actually assignable -- e.g., `reflect.Make*` functions return a pointer to the new object, but it is hidden behind an interface{} and this magic code extracts the actual underlying value



