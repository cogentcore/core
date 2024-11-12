// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plot

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32/minmax"
)

// Plotter is an interface that wraps the Plot method.
// Standard implementations of Plotter are in the [plots] package.
type Plotter interface {
	// Plot draws the data to the Plot Paint.
	Plot(pt *Plot)

	// UpdateRange updates the given ranges.
	UpdateRange(plt *Plot, xr, yr, zr *minmax.F64)

	// Data returns the data by roles for this plot, for both the original
	// data and the pixel-transformed X,Y coordinates for that data.
	// This allows a GUI interface to inspect data etc.
	Data() (data Data, pixX, pixY []float32)

	// Stylers returns the styler functions for this element.
	Stylers() *Stylers

	// ApplyStyle applies any stylers to this element,
	// first initializing from the given global plot style, which has
	// already been styled with defaults and all the plot element stylers.
	ApplyStyle(plotStyle *PlotStyle)
}

// PlotterType registers a Plotter so that it can be created with appropriate data.
type PlotterType struct {
	// Name of the plot type.
	Name string

	// Doc is the documentation for this Plotter.
	Doc string

	// Required Data roles for this plot. Data for these Roles must be provided.
	Required []Roles

	// Optional Data roles for this plot.
	Optional []Roles

	// New returns a new plotter of this type with given data in given roles.
	New func(data Data) Plotter
}

// PlotterName is the name of a specific plotter type.
type PlotterName string

// Plotters is the registry of [Plotter] types.
var Plotters = map[string]PlotterType{}

// RegisterPlotter registers a plotter type.
func RegisterPlotter(name, doc string, required, optional []Roles, newFun func(data Data) Plotter) {
	Plotters[name] = PlotterType{Name: name, Doc: doc, Required: required, Optional: optional, New: newFun}
}

// PlotterByType returns [PlotterType] info for a registered [Plotter]
// of given type name, e.g., "XY", "Bar" etc,
// Returns an error and nil if type name is not a registered type.
func PlotterByType(typeName string) (*PlotterType, error) {
	pt, ok := Plotters[typeName]
	if !ok {
		return nil, fmt.Errorf("plot.PlotterByType type name is not registered: %s", typeName)
	}
	return &pt, nil
}

// NewPlotter returns a new plotter of given type, e.g., "XY", "Bar" etc,
// for given data roles (which must include Required roles, and may include Optional ones).
// Logs an error and returns nil if type name is not a registered type.
func NewPlotter(typeName string, data Data) Plotter {
	pt, err := PlotterByType(typeName)
	if errors.Log(err) != nil {
		return nil
	}
	return pt.New(data)
}
