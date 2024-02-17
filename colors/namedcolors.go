// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on golang.org/x/image/colornames
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package colors

import (
	"image/color"
)

// Map contains named colors defined in the CSS spec.
var Map = map[string]color.RGBA{
	"aliceblue":            {0xf0, 0xf8, 0xff, 0xff}, // rgb(240, 248, 255)
	"antiquewhite":         {0xfa, 0xeb, 0xd7, 0xff}, // rgb(250, 235, 215)
	"aqua":                 {0x00, 0xff, 0xff, 0xff}, // rgb(0, 255, 255)
	"aquamarine":           {0x7f, 0xff, 0xd4, 0xff}, // rgb(127, 255, 212)
	"azure":                {0xf0, 0xff, 0xff, 0xff}, // rgb(240, 255, 255)
	"beige":                {0xf5, 0xf5, 0xdc, 0xff}, // rgb(245, 245, 220)
	"bisque":               {0xff, 0xe4, 0xc4, 0xff}, // rgb(255, 228, 196)
	"black":                {0x00, 0x00, 0x00, 0xff}, // rgb(0, 0, 0)
	"blanchedalmond":       {0xff, 0xeb, 0xcd, 0xff}, // rgb(255, 235, 205)
	"blue":                 {0x00, 0x00, 0xff, 0xff}, // rgb(0, 0, 255)
	"blueviolet":           {0x8a, 0x2b, 0xe2, 0xff}, // rgb(138, 43, 226)
	"brown":                {0xa5, 0x2a, 0x2a, 0xff}, // rgb(165, 42, 42)
	"burlywood":            {0xde, 0xb8, 0x87, 0xff}, // rgb(222, 184, 135)
	"cadetblue":            {0x5f, 0x9e, 0xa0, 0xff}, // rgb(95, 158, 160)
	"chartreuse":           {0x7f, 0xff, 0x00, 0xff}, // rgb(127, 255, 0)
	"chocolate":            {0xd2, 0x69, 0x1e, 0xff}, // rgb(210, 105, 30)
	"coral":                {0xff, 0x7f, 0x50, 0xff}, // rgb(255, 127, 80)
	"cornflowerblue":       {0x64, 0x95, 0xed, 0xff}, // rgb(100, 149, 237)
	"cornsilk":             {0xff, 0xf8, 0xdc, 0xff}, // rgb(255, 248, 220)
	"crimson":              {0xdc, 0x14, 0x3c, 0xff}, // rgb(220, 20, 60)
	"cyan":                 {0x00, 0xff, 0xff, 0xff}, // rgb(0, 255, 255)
	"darkblue":             {0x00, 0x00, 0x8b, 0xff}, // rgb(0, 0, 139)
	"darkcyan":             {0x00, 0x8b, 0x8b, 0xff}, // rgb(0, 139, 139)
	"darkgoldenrod":        {0xb8, 0x86, 0x0b, 0xff}, // rgb(184, 134, 11)
	"darkgray":             {0xa9, 0xa9, 0xa9, 0xff}, // rgb(169, 169, 169)
	"darkgreen":            {0x00, 0x64, 0x00, 0xff}, // rgb(0, 100, 0)
	"darkgrey":             {0xa9, 0xa9, 0xa9, 0xff}, // rgb(169, 169, 169)
	"darkkhaki":            {0xbd, 0xb7, 0x6b, 0xff}, // rgb(189, 183, 107)
	"darkmagenta":          {0x8b, 0x00, 0x8b, 0xff}, // rgb(139, 0, 139)
	"darkolivegreen":       {0x55, 0x6b, 0x2f, 0xff}, // rgb(85, 107, 47)
	"darkorange":           {0xff, 0x8c, 0x00, 0xff}, // rgb(255, 140, 0)
	"darkorchid":           {0x99, 0x32, 0xcc, 0xff}, // rgb(153, 50, 204)
	"darkred":              {0x8b, 0x00, 0x00, 0xff}, // rgb(139, 0, 0)
	"darksalmon":           {0xe9, 0x96, 0x7a, 0xff}, // rgb(233, 150, 122)
	"darkseagreen":         {0x8f, 0xbc, 0x8f, 0xff}, // rgb(143, 188, 143)
	"darkslateblue":        {0x48, 0x3d, 0x8b, 0xff}, // rgb(72, 61, 139)
	"darkslategray":        {0x2f, 0x4f, 0x4f, 0xff}, // rgb(47, 79, 79)
	"darkslategrey":        {0x2f, 0x4f, 0x4f, 0xff}, // rgb(47, 79, 79)
	"darkturquoise":        {0x00, 0xce, 0xd1, 0xff}, // rgb(0, 206, 209)
	"darkviolet":           {0x94, 0x00, 0xd3, 0xff}, // rgb(148, 0, 211)
	"deeppink":             {0xff, 0x14, 0x93, 0xff}, // rgb(255, 20, 147)
	"deepskyblue":          {0x00, 0xbf, 0xff, 0xff}, // rgb(0, 191, 255)
	"dimgray":              {0x69, 0x69, 0x69, 0xff}, // rgb(105, 105, 105)
	"dimgrey":              {0x69, 0x69, 0x69, 0xff}, // rgb(105, 105, 105)
	"dodgerblue":           {0x1e, 0x90, 0xff, 0xff}, // rgb(30, 144, 255)
	"firebrick":            {0xb2, 0x22, 0x22, 0xff}, // rgb(178, 34, 34)
	"floralwhite":          {0xff, 0xfa, 0xf0, 0xff}, // rgb(255, 250, 240)
	"forestgreen":          {0x22, 0x8b, 0x22, 0xff}, // rgb(34, 139, 34)
	"fuchsia":              {0xff, 0x00, 0xff, 0xff}, // rgb(255, 0, 255)
	"gainsboro":            {0xdc, 0xdc, 0xdc, 0xff}, // rgb(220, 220, 220)
	"ghostwhite":           {0xf8, 0xf8, 0xff, 0xff}, // rgb(248, 248, 255)
	"gold":                 {0xff, 0xd7, 0x00, 0xff}, // rgb(255, 215, 0)
	"goldenrod":            {0xda, 0xa5, 0x20, 0xff}, // rgb(218, 165, 32)
	"gray":                 {0x80, 0x80, 0x80, 0xff}, // rgb(128, 128, 128)
	"green":                {0x00, 0x80, 0x00, 0xff}, // rgb(0, 128, 0)
	"greenyellow":          {0xad, 0xff, 0x2f, 0xff}, // rgb(173, 255, 47)
	"grey":                 {0x80, 0x80, 0x80, 0xff}, // rgb(128, 128, 128)
	"honeydew":             {0xf0, 0xff, 0xf0, 0xff}, // rgb(240, 255, 240)
	"hotpink":              {0xff, 0x69, 0xb4, 0xff}, // rgb(255, 105, 180)
	"indianred":            {0xcd, 0x5c, 0x5c, 0xff}, // rgb(205, 92, 92)
	"indigo":               {0x4b, 0x00, 0x82, 0xff}, // rgb(75, 0, 130)
	"ivory":                {0xff, 0xff, 0xf0, 0xff}, // rgb(255, 255, 240)
	"khaki":                {0xf0, 0xe6, 0x8c, 0xff}, // rgb(240, 230, 140)
	"lavender":             {0xe6, 0xe6, 0xfa, 0xff}, // rgb(230, 230, 250)
	"lavenderblush":        {0xff, 0xf0, 0xf5, 0xff}, // rgb(255, 240, 245)
	"lawngreen":            {0x7c, 0xfc, 0x00, 0xff}, // rgb(124, 252, 0)
	"lemonchiffon":         {0xff, 0xfa, 0xcd, 0xff}, // rgb(255, 250, 205)
	"lightblue":            {0xad, 0xd8, 0xe6, 0xff}, // rgb(173, 216, 230)
	"lightcoral":           {0xf0, 0x80, 0x80, 0xff}, // rgb(240, 128, 128)
	"lightcyan":            {0xe0, 0xff, 0xff, 0xff}, // rgb(224, 255, 255)
	"lightgoldenrodyellow": {0xfa, 0xfa, 0xd2, 0xff}, // rgb(250, 250, 210)
	"lightgray":            {0xd3, 0xd3, 0xd3, 0xff}, // rgb(211, 211, 211)
	"lightgreen":           {0x90, 0xee, 0x90, 0xff}, // rgb(144, 238, 144)
	"lightgrey":            {0xd3, 0xd3, 0xd3, 0xff}, // rgb(211, 211, 211)
	"lightpink":            {0xff, 0xb6, 0xc1, 0xff}, // rgb(255, 182, 193)
	"lightsalmon":          {0xff, 0xa0, 0x7a, 0xff}, // rgb(255, 160, 122)
	"lightseagreen":        {0x20, 0xb2, 0xaa, 0xff}, // rgb(32, 178, 170)
	"lightskyblue":         {0x87, 0xce, 0xfa, 0xff}, // rgb(135, 206, 250)
	"lightslategray":       {0x77, 0x88, 0x99, 0xff}, // rgb(119, 136, 153)
	"lightslategrey":       {0x77, 0x88, 0x99, 0xff}, // rgb(119, 136, 153)
	"lightsteelblue":       {0xb0, 0xc4, 0xde, 0xff}, // rgb(176, 196, 222)
	"lightyellow":          {0xff, 0xff, 0xe0, 0xff}, // rgb(255, 255, 224)
	"lime":                 {0x00, 0xff, 0x00, 0xff}, // rgb(0, 255, 0)
	"limegreen":            {0x32, 0xcd, 0x32, 0xff}, // rgb(50, 205, 50)
	"linen":                {0xfa, 0xf0, 0xe6, 0xff}, // rgb(250, 240, 230)
	"magenta":              {0xff, 0x00, 0xff, 0xff}, // rgb(255, 0, 255)
	"maroon":               {0x80, 0x00, 0x00, 0xff}, // rgb(128, 0, 0)
	"mediumaquamarine":     {0x66, 0xcd, 0xaa, 0xff}, // rgb(102, 205, 170)
	"mediumblue":           {0x00, 0x00, 0xcd, 0xff}, // rgb(0, 0, 205)
	"mediumorchid":         {0xba, 0x55, 0xd3, 0xff}, // rgb(186, 85, 211)
	"mediumpurple":         {0x93, 0x70, 0xdb, 0xff}, // rgb(147, 112, 219)
	"mediumseagreen":       {0x3c, 0xb3, 0x71, 0xff}, // rgb(60, 179, 113)
	"mediumslateblue":      {0x7b, 0x68, 0xee, 0xff}, // rgb(123, 104, 238)
	"mediumspringgreen":    {0x00, 0xfa, 0x9a, 0xff}, // rgb(0, 250, 154)
	"mediumturquoise":      {0x48, 0xd1, 0xcc, 0xff}, // rgb(72, 209, 204)
	"mediumvioletred":      {0xc7, 0x15, 0x85, 0xff}, // rgb(199, 21, 133)
	"midnightblue":         {0x19, 0x19, 0x70, 0xff}, // rgb(25, 25, 112)
	"mintcream":            {0xf5, 0xff, 0xfa, 0xff}, // rgb(245, 255, 250)
	"mistyrose":            {0xff, 0xe4, 0xe1, 0xff}, // rgb(255, 228, 225)
	"moccasin":             {0xff, 0xe4, 0xb5, 0xff}, // rgb(255, 228, 181)
	"navajowhite":          {0xff, 0xde, 0xad, 0xff}, // rgb(255, 222, 173)
	"navy":                 {0x00, 0x00, 0x80, 0xff}, // rgb(0, 0, 128)
	"oldlace":              {0xfd, 0xf5, 0xe6, 0xff}, // rgb(253, 245, 230)
	"olive":                {0x80, 0x80, 0x00, 0xff}, // rgb(128, 128, 0)
	"olivedrab":            {0x6b, 0x8e, 0x23, 0xff}, // rgb(107, 142, 35)
	"orange":               {0xff, 0xa5, 0x00, 0xff}, // rgb(255, 165, 0)
	"orangered":            {0xff, 0x45, 0x00, 0xff}, // rgb(255, 69, 0)
	"orchid":               {0xda, 0x70, 0xd6, 0xff}, // rgb(218, 112, 214)
	"palegoldenrod":        {0xee, 0xe8, 0xaa, 0xff}, // rgb(238, 232, 170)
	"palegreen":            {0x98, 0xfb, 0x98, 0xff}, // rgb(152, 251, 152)
	"paleturquoise":        {0xaf, 0xee, 0xee, 0xff}, // rgb(175, 238, 238)
	"palevioletred":        {0xdb, 0x70, 0x93, 0xff}, // rgb(219, 112, 147)
	"papayawhip":           {0xff, 0xef, 0xd5, 0xff}, // rgb(255, 239, 213)
	"peachpuff":            {0xff, 0xda, 0xb9, 0xff}, // rgb(255, 218, 185)
	"peru":                 {0xcd, 0x85, 0x3f, 0xff}, // rgb(205, 133, 63)
	"pink":                 {0xff, 0xc0, 0xcb, 0xff}, // rgb(255, 192, 203)
	"plum":                 {0xdd, 0xa0, 0xdd, 0xff}, // rgb(221, 160, 221)
	"powderblue":           {0xb0, 0xe0, 0xe6, 0xff}, // rgb(176, 224, 230)
	"purple":               {0x80, 0x00, 0x80, 0xff}, // rgb(128, 0, 128)
	"rebeccapurple":        {0x66, 0x33, 0x99, 0xff}, // rgb(102, 51, 153)
	"red":                  {0xff, 0x00, 0x00, 0xff}, // rgb(255, 0, 0)
	"rosybrown":            {0xbc, 0x8f, 0x8f, 0xff}, // rgb(188, 143, 143)
	"royalblue":            {0x41, 0x69, 0xe1, 0xff}, // rgb(65, 105, 225)
	"saddlebrown":          {0x8b, 0x45, 0x13, 0xff}, // rgb(139, 69, 19)
	"salmon":               {0xfa, 0x80, 0x72, 0xff}, // rgb(250, 128, 114)
	"sandybrown":           {0xf4, 0xa4, 0x60, 0xff}, // rgb(244, 164, 96)
	"seagreen":             {0x2e, 0x8b, 0x57, 0xff}, // rgb(46, 139, 87)
	"seashell":             {0xff, 0xf5, 0xee, 0xff}, // rgb(255, 245, 238)
	"sienna":               {0xa0, 0x52, 0x2d, 0xff}, // rgb(160, 82, 45)
	"silver":               {0xc0, 0xc0, 0xc0, 0xff}, // rgb(192, 192, 192)
	"skyblue":              {0x87, 0xce, 0xeb, 0xff}, // rgb(135, 206, 235)
	"slateblue":            {0x6a, 0x5a, 0xcd, 0xff}, // rgb(106, 90, 205)
	"slategray":            {0x70, 0x80, 0x90, 0xff}, // rgb(112, 128, 144)
	"slategrey":            {0x70, 0x80, 0x90, 0xff}, // rgb(112, 128, 144)
	"snow":                 {0xff, 0xfa, 0xfa, 0xff}, // rgb(255, 250, 250)
	"springgreen":          {0x00, 0xff, 0x7f, 0xff}, // rgb(0, 255, 127)
	"steelblue":            {0x46, 0x82, 0xb4, 0xff}, // rgb(70, 130, 180)
	"tan":                  {0xd2, 0xb4, 0x8c, 0xff}, // rgb(210, 180, 140)
	"teal":                 {0x00, 0x80, 0x80, 0xff}, // rgb(0, 128, 128)
	"thistle":              {0xd8, 0xbf, 0xd8, 0xff}, // rgb(216, 191, 216)
	"tomato":               {0xff, 0x63, 0x47, 0xff}, // rgb(255, 99, 71)
	"turquoise":            {0x40, 0xe0, 0xd0, 0xff}, // rgb(64, 224, 208)
	"violet":               {0xee, 0x82, 0xee, 0xff}, // rgb(238, 130, 238)
	"wheat":                {0xf5, 0xde, 0xb3, 0xff}, // rgb(245, 222, 179)
	"white":                {0xff, 0xff, 0xff, 0xff}, // rgb(255, 255, 255)
	"whitesmoke":           {0xf5, 0xf5, 0xf5, 0xff}, // rgb(245, 245, 245)
	"yellow":               {0xff, 0xff, 0x00, 0xff}, // rgb(255, 255, 0)
	"yellowgreen":          {0x9a, 0xcd, 0x32, 0xff}, // rgb(154, 205, 50)
	"transparent":          {0, 0, 0, 0},             // rgb(0, 0, 0, 0)
}

// Names contains the color names defined in the CSS spec.
var Names = []string{
	"aliceblue",
	"antiquewhite",
	"aqua",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"black",
	"blanchedalmond",
	"blue",
	"blueviolet",
	"brown",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"crimson",
	"cyan",
	"darkblue",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkgreen",
	"darkgrey",
	"darkkhaki",
	"darkmagenta",
	"darkolivegreen",
	"darkorange",
	"darkorchid",
	"darkred",
	"darksalmon",
	"darkseagreen",
	"darkslateblue",
	"darkslategray",
	"darkslategrey",
	"darkturquoise",
	"darkviolet",
	"deeppink",
	"deepskyblue",
	"dimgray",
	"dimgrey",
	"dodgerblue",
	"firebrick",
	"floralwhite",
	"forestgreen",
	"fuchsia",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"gray",
	"green",
	"greenyellow",
	"grey",
	"honeydew",
	"hotpink",
	"indianred",
	"indigo",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgray",
	"lightgreen",
	"lightgrey",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightslategrey",
	"lightsteelblue",
	"lightyellow",
	"lime",
	"limegreen",
	"linen",
	"magenta",
	"maroon",
	"mediumaquamarine",
	"mediumblue",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mediumvioletred",
	"midnightblue",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"navy",
	"oldlace",
	"olive",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"purple",
	"rebeccapurple",
	"red",
	"rosybrown",
	"royalblue",
	"saddlebrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"sienna",
	"silver",
	"skyblue",
	"slateblue",
	"slategray",
	"slategrey",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"teal",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"white",
	"whitesmoke",
	"yellow",
	"yellowgreen",
	"transparent",
}

var (
	Aliceblue            = color.RGBA{R: 0xf0, G: 0xf8, B: 0xff, A: 0xff} // rgb(240, 248, 255)
	Antiquewhite         = color.RGBA{R: 0xfa, G: 0xeb, B: 0xd7, A: 0xff} // rgb(250, 235, 215)
	Aqua                 = color.RGBA{G: 0xff, B: 0xff, A: 0xff}          // rgb(0, 255, 255)
	Aquamarine           = color.RGBA{R: 0x7f, G: 0xff, B: 0xd4, A: 0xff} // rgb(127, 255, 212)
	Azure                = color.RGBA{R: 0xf0, G: 0xff, B: 0xff, A: 0xff} // rgb(240, 255, 255)
	Beige                = color.RGBA{R: 0xf5, G: 0xf5, B: 0xdc, A: 0xff} // rgb(245, 245, 220)
	Bisque               = color.RGBA{R: 0xff, G: 0xe4, B: 0xc4, A: 0xff} // rgb(255, 228, 196)
	Black                = color.RGBA{A: 0xff}                            // rgb(0, 0, 0)
	Blanchedalmond       = color.RGBA{R: 0xff, G: 0xeb, B: 0xcd, A: 0xff} // rgb(255, 235, 205)
	Blue                 = color.RGBA{B: 0xff, A: 0xff}                   // rgb(0, 0, 255)
	Blueviolet           = color.RGBA{R: 0x8a, G: 0x2b, B: 0xe2, A: 0xff} // rgb(138, 43, 226)
	Brown                = color.RGBA{R: 0xa5, G: 0x2a, B: 0x2a, A: 0xff} // rgb(165, 42, 42)
	Burlywood            = color.RGBA{R: 0xde, G: 0xb8, B: 0x87, A: 0xff} // rgb(222, 184, 135)
	Cadetblue            = color.RGBA{R: 0x5f, G: 0x9e, B: 0xa0, A: 0xff} // rgb(95, 158, 160)
	Chartreuse           = color.RGBA{R: 0x7f, G: 0xff, A: 0xff}          // rgb(127, 255, 0)
	Chocolate            = color.RGBA{R: 0xd2, G: 0x69, B: 0x1e, A: 0xff} // rgb(210, 105, 30)
	Coral                = color.RGBA{R: 0xff, G: 0x7f, B: 0x50, A: 0xff} // rgb(255, 127, 80)
	Cornflowerblue       = color.RGBA{R: 0x64, G: 0x95, B: 0xed, A: 0xff} // rgb(100, 149, 237)
	Cornsilk             = color.RGBA{R: 0xff, G: 0xf8, B: 0xdc, A: 0xff} // rgb(255, 248, 220)
	Crimson              = color.RGBA{R: 0xdc, G: 0x14, B: 0x3c, A: 0xff} // rgb(220, 20, 60)
	Cyan                 = color.RGBA{G: 0xff, B: 0xff, A: 0xff}          // rgb(0, 255, 255)
	Darkblue             = color.RGBA{B: 0x8b, A: 0xff}                   // rgb(0, 0, 139)
	Darkcyan             = color.RGBA{G: 0x8b, B: 0x8b, A: 0xff}          // rgb(0, 139, 139)
	Darkgoldenrod        = color.RGBA{R: 0xb8, G: 0x86, B: 0x0b, A: 0xff} // rgb(184, 134, 11)
	Darkgray             = color.RGBA{R: 0xa9, G: 0xa9, B: 0xa9, A: 0xff} // rgb(169, 169, 169)
	Darkgreen            = color.RGBA{G: 0x64, A: 0xff}                   // rgb(0, 100, 0)
	Darkgrey             = color.RGBA{R: 0xa9, G: 0xa9, B: 0xa9, A: 0xff} // rgb(169, 169, 169)
	Darkkhaki            = color.RGBA{R: 0xbd, G: 0xb7, B: 0x6b, A: 0xff} // rgb(189, 183, 107)
	Darkmagenta          = color.RGBA{R: 0x8b, B: 0x8b, A: 0xff}          // rgb(139, 0, 139)
	Darkolivegreen       = color.RGBA{R: 0x55, G: 0x6b, B: 0x2f, A: 0xff} // rgb(85, 107, 47)
	Darkorange           = color.RGBA{R: 0xff, G: 0x8c, A: 0xff}          // rgb(255, 140, 0)
	Darkorchid           = color.RGBA{R: 0x99, G: 0x32, B: 0xcc, A: 0xff} // rgb(153, 50, 204)
	Darkred              = color.RGBA{R: 0x8b, A: 0xff}                   // rgb(139, 0, 0)
	Darksalmon           = color.RGBA{R: 0xe9, G: 0x96, B: 0x7a, A: 0xff} // rgb(233, 150, 122)
	Darkseagreen         = color.RGBA{R: 0x8f, G: 0xbc, B: 0x8f, A: 0xff} // rgb(143, 188, 143)
	Darkslateblue        = color.RGBA{R: 0x48, G: 0x3d, B: 0x8b, A: 0xff} // rgb(72, 61, 139)
	Darkslategray        = color.RGBA{R: 0x2f, G: 0x4f, B: 0x4f, A: 0xff} // rgb(47, 79, 79)
	Darkslategrey        = color.RGBA{R: 0x2f, G: 0x4f, B: 0x4f, A: 0xff} // rgb(47, 79, 79)
	Darkturquoise        = color.RGBA{G: 0xce, B: 0xd1, A: 0xff}          // rgb(0, 206, 209)
	Darkviolet           = color.RGBA{R: 0x94, B: 0xd3, A: 0xff}          // rgb(148, 0, 211)
	Deeppink             = color.RGBA{R: 0xff, G: 0x14, B: 0x93, A: 0xff} // rgb(255, 20, 147)
	Deepskyblue          = color.RGBA{G: 0xbf, B: 0xff, A: 0xff}          // rgb(0, 191, 255)
	Dimgray              = color.RGBA{R: 0x69, G: 0x69, B: 0x69, A: 0xff} // rgb(105, 105, 105)
	Dimgrey              = color.RGBA{R: 0x69, G: 0x69, B: 0x69, A: 0xff} // rgb(105, 105, 105)
	Dodgerblue           = color.RGBA{R: 0x1e, G: 0x90, B: 0xff, A: 0xff} // rgb(30, 144, 255)
	Firebrick            = color.RGBA{R: 0xb2, G: 0x22, B: 0x22, A: 0xff} // rgb(178, 34, 34)
	Floralwhite          = color.RGBA{R: 0xff, G: 0xfa, B: 0xf0, A: 0xff} // rgb(255, 250, 240)
	Forestgreen          = color.RGBA{R: 0x22, G: 0x8b, B: 0x22, A: 0xff} // rgb(34, 139, 34)
	Fuchsia              = color.RGBA{R: 0xff, B: 0xff, A: 0xff}          // rgb(255, 0, 255)
	Gainsboro            = color.RGBA{R: 0xdc, G: 0xdc, B: 0xdc, A: 0xff} // rgb(220, 220, 220)
	Ghostwhite           = color.RGBA{R: 0xf8, G: 0xf8, B: 0xff, A: 0xff} // rgb(248, 248, 255)
	Gold                 = color.RGBA{R: 0xff, G: 0xd7, A: 0xff}          // rgb(255, 215, 0)
	Goldenrod            = color.RGBA{R: 0xda, G: 0xa5, B: 0x20, A: 0xff} // rgb(218, 165, 32)
	Gray                 = color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff} // rgb(128, 128, 128)
	Green                = color.RGBA{G: 0x80, A: 0xff}                   // rgb(0, 128, 0)
	Greenyellow          = color.RGBA{R: 0xad, G: 0xff, B: 0x2f, A: 0xff} // rgb(173, 255, 47)
	Grey                 = color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff} // rgb(128, 128, 128)
	Honeydew             = color.RGBA{R: 0xf0, G: 0xff, B: 0xf0, A: 0xff} // rgb(240, 255, 240)
	Hotpink              = color.RGBA{R: 0xff, G: 0x69, B: 0xb4, A: 0xff} // rgb(255, 105, 180)
	Indianred            = color.RGBA{R: 0xcd, G: 0x5c, B: 0x5c, A: 0xff} // rgb(205, 92, 92)
	Indigo               = color.RGBA{R: 0x4b, B: 0x82, A: 0xff}          // rgb(75, 0, 130)
	Ivory                = color.RGBA{R: 0xff, G: 0xff, B: 0xf0, A: 0xff} // rgb(255, 255, 240)
	Khaki                = color.RGBA{R: 0xf0, G: 0xe6, B: 0x8c, A: 0xff} // rgb(240, 230, 140)
	Lavender             = color.RGBA{R: 0xe6, G: 0xe6, B: 0xfa, A: 0xff} // rgb(230, 230, 250)
	Lavenderblush        = color.RGBA{R: 0xff, G: 0xf0, B: 0xf5, A: 0xff} // rgb(255, 240, 245)
	Lawngreen            = color.RGBA{R: 0x7c, G: 0xfc, A: 0xff}          // rgb(124, 252, 0)
	Lemonchiffon         = color.RGBA{R: 0xff, G: 0xfa, B: 0xcd, A: 0xff} // rgb(255, 250, 205)
	Lightblue            = color.RGBA{R: 0xad, G: 0xd8, B: 0xe6, A: 0xff} // rgb(173, 216, 230)
	Lightcoral           = color.RGBA{R: 0xf0, G: 0x80, B: 0x80, A: 0xff} // rgb(240, 128, 128)
	Lightcyan            = color.RGBA{R: 0xe0, G: 0xff, B: 0xff, A: 0xff} // rgb(224, 255, 255)
	Lightgoldenrodyellow = color.RGBA{R: 0xfa, G: 0xfa, B: 0xd2, A: 0xff} // rgb(250, 250, 210)
	Lightgray            = color.RGBA{R: 0xd3, G: 0xd3, B: 0xd3, A: 0xff} // rgb(211, 211, 211)
	Lightgreen           = color.RGBA{R: 0x90, G: 0xee, B: 0x90, A: 0xff} // rgb(144, 238, 144)
	Lightgrey            = color.RGBA{R: 0xd3, G: 0xd3, B: 0xd3, A: 0xff} // rgb(211, 211, 211)
	Lightpink            = color.RGBA{R: 0xff, G: 0xb6, B: 0xc1, A: 0xff} // rgb(255, 182, 193)
	Lightsalmon          = color.RGBA{R: 0xff, G: 0xa0, B: 0x7a, A: 0xff} // rgb(255, 160, 122)
	Lightseagreen        = color.RGBA{R: 0x20, G: 0xb2, B: 0xaa, A: 0xff} // rgb(32, 178, 170)
	Lightskyblue         = color.RGBA{R: 0x87, G: 0xce, B: 0xfa, A: 0xff} // rgb(135, 206, 250)
	Lightslategray       = color.RGBA{R: 0x77, G: 0x88, B: 0x99, A: 0xff} // rgb(119, 136, 153)
	Lightslategrey       = color.RGBA{R: 0x77, G: 0x88, B: 0x99, A: 0xff} // rgb(119, 136, 153)
	Lightsteelblue       = color.RGBA{R: 0xb0, G: 0xc4, B: 0xde, A: 0xff} // rgb(176, 196, 222)
	Lightyellow          = color.RGBA{R: 0xff, G: 0xff, B: 0xe0, A: 0xff} // rgb(255, 255, 224)
	Lime                 = color.RGBA{G: 0xff, A: 0xff}                   // rgb(0, 255, 0)
	Limegreen            = color.RGBA{R: 0x32, G: 0xcd, B: 0x32, A: 0xff} // rgb(50, 205, 50)
	Linen                = color.RGBA{R: 0xfa, G: 0xf0, B: 0xe6, A: 0xff} // rgb(250, 240, 230)
	Magenta              = color.RGBA{R: 0xff, B: 0xff, A: 0xff}          // rgb(255, 0, 255)
	Maroon               = color.RGBA{R: 0x80, A: 0xff}                   // rgb(128, 0, 0)
	Mediumaquamarine     = color.RGBA{R: 0x66, G: 0xcd, B: 0xaa, A: 0xff} // rgb(102, 205, 170)
	Mediumblue           = color.RGBA{B: 0xcd, A: 0xff}                   // rgb(0, 0, 205)
	Mediumorchid         = color.RGBA{R: 0xba, G: 0x55, B: 0xd3, A: 0xff} // rgb(186, 85, 211)
	Mediumpurple         = color.RGBA{R: 0x93, G: 0x70, B: 0xdb, A: 0xff} // rgb(147, 112, 219)
	Mediumseagreen       = color.RGBA{R: 0x3c, G: 0xb3, B: 0x71, A: 0xff} // rgb(60, 179, 113)
	Mediumslateblue      = color.RGBA{R: 0x7b, G: 0x68, B: 0xee, A: 0xff} // rgb(123, 104, 238)
	Mediumspringgreen    = color.RGBA{G: 0xfa, B: 0x9a, A: 0xff}          // rgb(0, 250, 154)
	Mediumturquoise      = color.RGBA{R: 0x48, G: 0xd1, B: 0xcc, A: 0xff} // rgb(72, 209, 204)
	Mediumvioletred      = color.RGBA{R: 0xc7, G: 0x15, B: 0x85, A: 0xff} // rgb(199, 21, 133)
	Midnightblue         = color.RGBA{R: 0x19, G: 0x19, B: 0x70, A: 0xff} // rgb(25, 25, 112)
	Mintcream            = color.RGBA{R: 0xf5, G: 0xff, B: 0xfa, A: 0xff} // rgb(245, 255, 250)
	Mistyrose            = color.RGBA{R: 0xff, G: 0xe4, B: 0xe1, A: 0xff} // rgb(255, 228, 225)
	Moccasin             = color.RGBA{R: 0xff, G: 0xe4, B: 0xb5, A: 0xff} // rgb(255, 228, 181)
	Navajowhite          = color.RGBA{R: 0xff, G: 0xde, B: 0xad, A: 0xff} // rgb(255, 222, 173)
	Navy                 = color.RGBA{B: 0x80, A: 0xff}                   // rgb(0, 0, 128)
	Oldlace              = color.RGBA{R: 0xfd, G: 0xf5, B: 0xe6, A: 0xff} // rgb(253, 245, 230)
	Olive                = color.RGBA{R: 0x80, G: 0x80, A: 0xff}          // rgb(128, 128, 0)
	Olivedrab            = color.RGBA{R: 0x6b, G: 0x8e, B: 0x23, A: 0xff} // rgb(107, 142, 35)
	Orange               = color.RGBA{R: 0xff, G: 0xa5, A: 0xff}          // rgb(255, 165, 0)
	Orangered            = color.RGBA{R: 0xff, G: 0x45, A: 0xff}          // rgb(255, 69, 0)
	Orchid               = color.RGBA{R: 0xda, G: 0x70, B: 0xd6, A: 0xff} // rgb(218, 112, 214)
	Palegoldenrod        = color.RGBA{R: 0xee, G: 0xe8, B: 0xaa, A: 0xff} // rgb(238, 232, 170)
	Palegreen            = color.RGBA{R: 0x98, G: 0xfb, B: 0x98, A: 0xff} // rgb(152, 251, 152)
	Paleturquoise        = color.RGBA{R: 0xaf, G: 0xee, B: 0xee, A: 0xff} // rgb(175, 238, 238)
	Palevioletred        = color.RGBA{R: 0xdb, G: 0x70, B: 0x93, A: 0xff} // rgb(219, 112, 147)
	Papayawhip           = color.RGBA{R: 0xff, G: 0xef, B: 0xd5, A: 0xff} // rgb(255, 239, 213)
	Peachpuff            = color.RGBA{R: 0xff, G: 0xda, B: 0xb9, A: 0xff} // rgb(255, 218, 185)
	Peru                 = color.RGBA{R: 0xcd, G: 0x85, B: 0x3f, A: 0xff} // rgb(205, 133, 63)
	Pink                 = color.RGBA{R: 0xff, G: 0xc0, B: 0xcb, A: 0xff} // rgb(255, 192, 203)
	Plum                 = color.RGBA{R: 0xdd, G: 0xa0, B: 0xdd, A: 0xff} // rgb(221, 160, 221)
	Powderblue           = color.RGBA{R: 0xb0, G: 0xe0, B: 0xe6, A: 0xff} // rgb(176, 224, 230)
	Purple               = color.RGBA{R: 0x80, B: 0x80, A: 0xff}          // rgb(128, 0, 128)
	Rebeccapurple        = color.RGBA{R: 0x66, G: 0x33, B: 0x99, A: 0xff} // rgb(102, 51, 153)
	Red                  = color.RGBA{R: 0xff, A: 0xff}                   // rgb(255, 0, 0)
	Rosybrown            = color.RGBA{R: 0xbc, G: 0x8f, B: 0x8f, A: 0xff} // rgb(188, 143, 143)
	Royalblue            = color.RGBA{R: 0x41, G: 0x69, B: 0xe1, A: 0xff} // rgb(65, 105, 225)
	Saddlebrown          = color.RGBA{R: 0x8b, G: 0x45, B: 0x13, A: 0xff} // rgb(139, 69, 19)
	Salmon               = color.RGBA{R: 0xfa, G: 0x80, B: 0x72, A: 0xff} // rgb(250, 128, 114)
	Sandybrown           = color.RGBA{R: 0xf4, G: 0xa4, B: 0x60, A: 0xff} // rgb(244, 164, 96)
	Seagreen             = color.RGBA{R: 0x2e, G: 0x8b, B: 0x57, A: 0xff} // rgb(46, 139, 87)
	Seashell             = color.RGBA{R: 0xff, G: 0xf5, B: 0xee, A: 0xff} // rgb(255, 245, 238)
	Sienna               = color.RGBA{R: 0xa0, G: 0x52, B: 0x2d, A: 0xff} // rgb(160, 82, 45)
	Silver               = color.RGBA{R: 0xc0, G: 0xc0, B: 0xc0, A: 0xff} // rgb(192, 192, 192)
	Skyblue              = color.RGBA{R: 0x87, G: 0xce, B: 0xeb, A: 0xff} // rgb(135, 206, 235)
	Slateblue            = color.RGBA{R: 0x6a, G: 0x5a, B: 0xcd, A: 0xff} // rgb(106, 90, 205)
	Slategray            = color.RGBA{R: 0x70, G: 0x80, B: 0x90, A: 0xff} // rgb(112, 128, 144)
	Slategrey            = color.RGBA{R: 0x70, G: 0x80, B: 0x90, A: 0xff} // rgb(112, 128, 144)
	Snow                 = color.RGBA{R: 0xff, G: 0xfa, B: 0xfa, A: 0xff} // rgb(255, 250, 250)
	Springgreen          = color.RGBA{G: 0xff, B: 0x7f, A: 0xff}          // rgb(0, 255, 127)
	Steelblue            = color.RGBA{R: 0x46, G: 0x82, B: 0xb4, A: 0xff} // rgb(70, 130, 180)
	Tan                  = color.RGBA{R: 0xd2, G: 0xb4, B: 0x8c, A: 0xff} // rgb(210, 180, 140)
	Teal                 = color.RGBA{G: 0x80, B: 0x80, A: 0xff}          // rgb(0, 128, 128)
	Thistle              = color.RGBA{R: 0xd8, G: 0xbf, B: 0xd8, A: 0xff} // rgb(216, 191, 216)
	Tomato               = color.RGBA{R: 0xff, G: 0x63, B: 0x47, A: 0xff} // rgb(255, 99, 71)
	Turquoise            = color.RGBA{R: 0x40, G: 0xe0, B: 0xd0, A: 0xff} // rgb(64, 224, 208)
	Violet               = color.RGBA{R: 0xee, G: 0x82, B: 0xee, A: 0xff} // rgb(238, 130, 238)
	Wheat                = color.RGBA{R: 0xf5, G: 0xde, B: 0xb3, A: 0xff} // rgb(245, 222, 179)
	White                = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff} // rgb(255, 255, 255)
	Whitesmoke           = color.RGBA{R: 0xf5, G: 0xf5, B: 0xf5, A: 0xff} // rgb(245, 245, 245)
	Yellow               = color.RGBA{R: 0xff, G: 0xff, A: 0xff}          // rgb(255, 255, 0)
	Yellowgreen          = color.RGBA{R: 0x9a, G: 0xcd, B: 0x32, A: 0xff} // rgb(154, 205, 50)
	Transparent          = color.RGBA{}                                   // rgb(0, 0, 0, 0)
)
