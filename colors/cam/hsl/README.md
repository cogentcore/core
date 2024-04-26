# hsl: Hue, Saturation and Lightness

HSL is a standard color model in wide use.  [HCT](../hct) produces better psychophysics for actual color perception, but is more computationally expensive.

* Hue [0..360]
* Saturation [0..1]
* Luminance (lightness) [0..1]


## HSL vs. HCT Colorspaces

Here is HSL:

![hsl colorspace](testdata/hslspace.png)

vs. HCT:

![hct colorspace](../hct/testdata/hctspace.png)

You can see that HSL has much greater variability in the brightness of the saturated colors, the progression of saturation, and the distribution of hues (yellow / green takes up a large space while red-orange goes by quickly).


