// Package directive implements simple, standardized, and scalable parsing of Go comment directives.
package directive

import "strings"

// Directive represents a comment directive
// that has been parsed or created in code.
type Directive struct {

	// Source is the source string of the
	// comment directive.
	Source string

	// Tool is the name of the tool that
	// the directive is for.
	Tool string

	// Directive is the actual directive
	// string that is placed after the
	// name of the tool and a colon.
	Directive string

	// Args are the positional arguments
	// passed to the directive
	Args []string

	// Props are a map of key-value
	// properties given in the
	// form `key=value`.
	Props map[string]string
}

// Parse parses the given comment string and returns
// any [Directive] inside it. It also returns whether
// it found such a directive (has), and any error that
// occurred while parsing the directive (err). A lack of a
// directive does not result in an error, just a false
// has value. Directives are of the form:
// `//tool:directive arg0 key0=value0 arg1 key1=value1`
// (the positional arguments and key-value arguments can
// be in any order).
func Parse(comment string) (dir Directive, has bool, err error) {
	before, after, found := strings.Cut(comment, ":")
	if !found {
		return Directive{}, false, nil
	}
	dir.Source = comment
	dir.Tool = before
	fields := strings.Fields(after)
	for _, field := range fields {

	}
}
