# hct: Hue, Chroma, and Tone

A color system built using CAM16 hue and chroma, and L* (lightness) from the L*a*b* color space, providing a perceptually accurate color measurement system that can also accurately render what colors will appear as in different lighting environments.

Using L* creates a link between the color system, contrast, and thus accessibility. Contrast ratio depends on relative luminance, or Y in the XYZ color space. L*, or perceptual luminance can be calculated from Y.

Unlike Y, L* is linear to human perception, allowing trivial creation of accurate color tones.

Unlike contrast ratio, measuring contrast in L* is linear, and simple to calculate. A difference of 40 in HCT tone guarantees a contrast ratio >= 3.0, and a difference of 50 guarantees a contrast ratio >= 4.5.



