# gti

Package GTI provides general purpose type information for Go types, methods, functions and variables

# key questions:

* separate registries?  YES -- keeps it type specific

* TypeRegistry
* FuncRegistry

not sure about these:

* VarRegistry
* ConstRegistry

# Key functionality

* generate tooltips for fields in structs

* generate toolbar props for gogi methodview to generate greasi toolbars for arbitrary App types, and for Ki types.

* in grease, command method docs and config field docs

* in Ki (optional) generate a new token of given type by name (for Unmarshal)

* Ki needs NameAndType, need to be able to use Type token to specify type.  Also for making a new child of given type.

# notes

* GTI is NOT Go ast because it doesn't store Type info for Fields, methods, etc.  There is no assumption that all types are processed -- only designated types.  It only records names, comments and directives.


