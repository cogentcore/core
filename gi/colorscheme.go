// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "goki.dev/colors/matcolor"

// ColorScheme is the current color scheme
// that is used to style the app. It should
// not be set by end-user code, as it is set
// automatically from the user's preferences and
// [ColorSchemes]. You should set [ColorSchemes]
// to customize the color scheme of your app.
var ColorScheme matcolor.Scheme
