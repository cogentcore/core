# cie

CIE is the International Commission on Illumination which establishes standards for representing color information.

This package contains standard values and routines for converting between standard RGB color space (sRGB), and the the CIE standard color spaces of XYZ and L*a*b* (LAB).

# XYZ color space (1931): standard color basis space

https://en.wikipedia.org/wiki/CIE_1931_color_space

* `Y` is the luminance (overall brightness) = 0.2 red + 0.7 green + 0.07 blue
* `Z` is purely the short wavelength (blue) = 0.02 red + 0.1 green + 0.95 blue
* `X` is a mix of the three CIE LMS cone responses chosen to be nonnegative: 1.9 long (red), -1.1 medium (green), and 0.2 short = 0.4 red + 0.36 green + 0.18 blue

The unit of the tristimulus values X, Y, and Z is often arbitrarily chosen so that Y = 1 or Y = 100 is the brightest white that a color display supports. In this case, the Y value is known as the relative luminance. The corresponding whitepoint values for X and Z can then be inferred using the standard illuminants.

# LAB  L*a*b* color space

https://en.wikipedia.org/wiki/CIELAB_color_space

* `lightness L*` defines black at 0 and white at 100.
* `a*` is relative to the green–magenta opponent colors, with negative values toward green and positive values toward magenta.
* `b*` represents the blue–yellow opponents, with negative numbers toward blue and positive toward yellow.

The a* and b* axes are unbounded and depending on the reference white they can easily exceed ±150 to cover the human gamut. Nevertheless, software implementations often clamp these values for practical reasons. For instance, if integer math is being used it is common to clamp a* and b* in the range of −128 to 127.

CIELAB is calculated relative to a reference white, for which the CIE recommends the use of CIE Standard illuminant D65.  D65 is used in the vast majority industries and applications, with the notable exception being the printing industry which uses D50.


