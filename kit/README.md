# kit

[![Go Report Card](https://goreportcard.com/badge/github.com/rcoreilly/goki/ki/kit)](https://goreportcard.com/report/github.com/rcoreilly/goki/ki/kit)
[![GoDoc](https://godoc.org/github.com/rcoreilly/goki/ki/kit?status.svg)](http://godoc.org/github.com/rcoreilly/goki/ki/kit)


Package kit provides a `TypeRegistry` for associating string names with
`reflect.Type` values, to allow dynamic marshaling of structs, and also
bidirectional string conversion of const int iota (enum) types.  It is used
by the GoKi ki system, hence the kit (ki types) name.

To register a new type, add:

``` go
var KiT_TypeName = kit.Types.AddType(&TypeName{}, [props|nil])
```

where the props is a `map[string]interface{}` of optional properties that can
be associated with the type -- this is used in the GoGi graphical interface
system for example to color objects of different types using the
background-color property.  `KiT_TypeName` variable can be conveniently used
wherever a reflect.Type of that type is needed.

The `kit.Type` struct provides JSON and XML Marshal / Unmarshal functions for
saving / loading reflect.Type using registrered type names.

Also provided are robust `interface{}`-based type conversion routines in
`convert.go` that are useful in more lax user-interface contexts where
"common sense" conversions between strings, numbers etc are useful
