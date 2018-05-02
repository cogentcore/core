# kit

[![Go Report Card](https://goreportcard.com/badge/github.com/goki/goki/ki/kit)](https://goreportcard.com/report/github.com/goki/goki/ki/kit)
[![GoDoc](https://godoc.org/github.com/goki/goki/ki/kit?status.svg)](http://godoc.org/github.com/goki/goki/ki/kit)

Package `kit1 provides various reflect type functions for GoKi system, including:

* `kit.TypeRegistry (types.go)` for associating string names with
`reflect.Type` values, to allow dynamic marshaling of structs, and also
bidirectional string conversion of const int iota (enum) types.  It is used
by the GoKi ki system, hence the kit (ki types) name.

To register a new type, add:

``` go
var KiT_TypeName = kit.Types.AddType(&TypeName{}, [props|nil])
```

where the props is a map[string]interface{} of optional properties that can
be associated with the type -- this is used in the GoGi graphical interface
system for example to color objects of different types using the
background-color property.  KiT_TypeName variable can be conveniently used
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
