// Code generated by "core generate"; DO NOT EDIT.

package enumgen

import (
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/enums/enumgen.Config", IDName: "config", Doc: "Config contains the configuration information\nused by enumgen", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Dir", Doc: "the source directory to run enumgen on (can be set to multiple through paths like ./...)"}, {Name: "Output", Doc: "the output file location relative to the package on which enumgen is being called"}, {Name: "Transform", Doc: "if specified, the enum item transformation method (upper, lower, snake, SNAKE, kebab, KEBAB,\ncamel, lower-camel, title, sentence, first, first-upper, or first-lower)"}, {Name: "TrimPrefix", Doc: "if specified, a comma-separated list of prefixes to trim from each item"}, {Name: "AddPrefix", Doc: "if specified, the prefix to add to each item"}, {Name: "LineComment", Doc: "whether to use line comment text as printed text when present"}, {Name: "AcceptLower", Doc: "whether to accept lowercase versions of enum names in SetString"}, {Name: "IsValid", Doc: "whether to generate a method returning whether a value is\na valid option for its enum type; this must also be set for\nany base enum type being extended"}, {Name: "Text", Doc: "whether to generate text marshaling methods"}, {Name: "SQL", Doc: "whether to generate methods that implement the SQL Scanner and Valuer interfaces"}, {Name: "GQL", Doc: "whether to generate GraphQL marshaling methods for gqlgen"}, {Name: "Extend", Doc: "whether to allow enums to extend other enums; this should be on in almost all circumstances,\nbut can be turned off for specific enum types that extend non-enum types"}}})
