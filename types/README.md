# types

Package types provides type information for Go types, methods, and functions.

# Key functionality

* Generate tooltips for fields in forms

* Generate tooltips for functions in func buttons

* In cli, generate command function docs and config field docs

* In tree (optional) generate a new token of given type by name

* Package tree needs type variables when making children from serialized data

# Notes

* It is NOT Go ast because it doesn't store type info for fields, methods, etc.  There is no assumption that all types are processed, only designated types.  It only records names, comments and directives.
