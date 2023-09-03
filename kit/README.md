# kit

[![Go Report Card](https://goreportcard.com/badge/goki.dev/ki/v2/kit)](https://goreportcard.com/report/goki.dev/ki/v2/kit)
[![GoDoc](https://godoc.org/goki.dev/ki/v2/kit?status.svg)](http://godoc.org/goki.dev/ki/v2/kit)

Package `kit` provides various reflect type functions for GoKi system (KiType = KiT = kit -- also a bit of a "kit" collection of low-level system functions), including:

* `kit.TypeRegistry (types.go)` for associating string names with
`reflect.Type` values, to allow dynamic marshaling of structs, and also
bidirectional string conversion of const int iota (enum) types.  It is used
by the GoKi ki system, hence the kit (ki types) name.

To register a new type, add:

```Go
var TypeTypeName = kit.Types.AddType(&TypeName{}, [props|nil])
```

where the props is a `map[string]interface{}` of optional properties that can
be associated with the type -- this is used in the GoGi graphical interface
system for example to color objects of different types using the
background-color property.  TypeTypeName variable can be conveniently used
wherever a reflect.Type of that type is needed.

* `kit.EnumRegistry (enums.go)` that registers constant int iota (aka enum) types, and
provides general conversion utilities to / from string, int64, general
properties associated with enum types, and deals with bit flags

* `kit.Type (type.go)` struct provides JSON and XML Marshal / Unmarshal functions for
saving / loading reflect.Type using registrered type names.

* `convert.go`: robust interface{}-based type conversion routines that are
useful in more lax user-interface contexts where "common sense" conversions
between strings, numbers etc are useful

* `embeds.go`: various functions for managing embedded struct types, e.g.,
determining if a given type embeds another type (directly or indirectly),
and iterating over fields to flatten the otherwise nested nature of the
field encoding in embedded types.
