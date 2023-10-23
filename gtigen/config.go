// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import "text/template"

// Config contains the configuration information
// used by gtigen
//
//gti:add
type Config struct {

	// the source directory to run gtigen on (can be set to multiple through paths like ./...)
	Dir string `def:"." posarg:"0" required:"-"`

	// the output file location relative to the package on which gtigen is being called
	Output string `def:"gtigen.go"`

	// whether to add types to gtigen by default
	AddTypes bool

	// whether to add methods to gtigen by default
	AddMethods bool

	// whether to add functions to gtigen by default
	AddFuncs bool

	// A map of configs keyed by fully-qualified interface type names; if a type implements the interface, the config will be applied to it.
	// Note: the package gtigen is run on must explicitly reference this interface at some point for this to work; adding a simple
	// `var _ MyInterface = (*MyType)(nil)` statement to check for interface implementation is an easy way to accomplish that.
	// Note: gtigen will still succeed if it can not find one of the interfaces specified here in order to allow it to work generically across multiple directories; you can use the -v flag to get log warnings about this if you suspect that it is not finding interfaces when it should.
	InterfaceConfigs map[string]*Config

	// whether to generate an instance of the type(s)
	Instance bool

	// whether to generate a global type variable of the form 'TypeNameType'
	TypeVar bool

	// Whether to generate chaining `Set*` methods for each field of each type (eg: "SetText" for field "Text").
	// If this is set to true, then you can add `set:"-"` struct tags to individual fields
	// to prevent Set methods being generated for them.
	Setters bool

	// TODO: should this be called TypeTemplates and should there be a Func/Method Templates?

	// a slice of templates to execute on each type being added; the template data is of the type gtigen.Type
	Templates []*template.Template
}
