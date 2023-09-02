// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/matcolor"

// ColorScheme is the current color scheme
// that is used to style the app. It should
// not be set by end-user code, as it is set
// automatically from the user's preferences and
// [ColorSchemes]. You should set [ColorSchemes]
// to customize the color scheme of your app.
var ColorScheme matcolor.Scheme

//go:generate goki colorgen colors.xml -p gi -c "// ColorSchemes contains the color schemes used to style\n// the app. You should set them in your mainrun funtion\n// if you want to change the color schemes.\n// [ColorScheme] is set based on ColorSchemes and\n// the user preferences. The default color schemes\n// are generated through the \"goki colorgen\" command,\n// and you should use the same command to generate your\n// custom color scheme. You should not directly access\n// ColorSchemes when styling things; instead, you should\n// access [ColorScheme]."
