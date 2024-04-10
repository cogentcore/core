# gti

Package GTI provides general purpose type information for Go types, methods, functions and variables.

# Key functionality

* generate tooltips for fields in struct views

* generate tooltips for functions in func buttons

* in grease, command method docs and config field docs

* in tree (optional) generate a new token of given type by name (for Unmarshal)

* tree needs NameAndType, need to be able to use Type token to specify type.  Also for making a new child of given type.

# Notes

* GTI is NOT Go ast because it doesn't store Type info for Fields, methods, etc.  There is no assumption that all types are processed -- only designated types.  It only records names, comments and directives.


