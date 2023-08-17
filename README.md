# cam: Color Appearance Models

The color axes in the human brain are Red-Green and Blue-Yellow. In color speak Red = Long, Green = Medium, Blue = Short wavelengths, or the [lms](lms) space, corresponding to the color absorption profile of cones in the fovea of the retina.

# cie: International Commission on Illumination

See [cie](cie) for standard color spaces and values defined by the International Commission on Illumination (CIE), including [XYZ](https://en.wikipedia.org/wiki/CIE_1931_color_space) and [LAB](https://en.wikipedia.org/wiki/CIELAB_color_space) (L\*a\*b\*) which can be easily computed from corresponding standard RGB (sRGB) values.

RGB as displayed on a computer monitor is typically "gamma corrected" to compensate for the luminance properties of the display, so this gamma correction must be removed before using RGB values to covert into other CIE spaces.

* Here’s what is implemented: 

* paper for this: Moroney et al., 2002

* Color spaces based on LMS cones: https://en.wikipedia.org/wiki/LMS_color_space

* Standard CIE 1931 XYZ color space: 

* Standard color checker: https://en.wikipedia.org/wiki/ColorChecker

# CAM02, CAM16

Robert W.G. Hunt established many of the key principles of color appearance models, as attested by the many references in [Helwig & Fairchild (2022)](#references) (HF22), which is a particularly good reference for actually explaining things in plain English.  [Moroney, N., Fairchild, M. D., Hunt, R. W. G., Li, C., Luo, M. R., & Newman, T. (2002)](#references) established the CIE CAM02 reference model, which set the standard for many years, until being revised in the CAM16 model.  Here's the HF22 explanation of the key factors in CAM02 and CAM16:

> **Brightness** is the perceptual attribute by which a light source or reflective surface appears to emit or reflect more or less light.  **Lightness** is the brightness of a stimulus relative to the brightness of a white-appearing stimulus in a similarly illuminated area, also known as the reference white.  While the brightness of stimuli has a general positive correlation with the amount of light they emit or reflect, there is no simple relationship between the amount of light emitted by a stimulus and its brightness and lightness. For instance, stimuli with greater purity appear brighter than stimuli with less purity if they are of the same luminance (known as the Helmholtz–Kohlrausch Effect).

> The perceptual attribute **colorfulness** describes the absolute chromatic intensity of a visual stimulus. Chroma and saturation are relative measures of colorfulness; **chroma** is defined as the colorfulness of a stimulus relative to the brightness of similarly illuminated white and **saturation** is defined as the colorfulness of a stimulus relative to its own brightness.

## CAM02

See [cam02](cam02) for implementation, and [Moroney et al, 2002](#references) for description, along with this [wikipedia](https://en.wikipedia.org/wiki/CIECAM02) page.

We implement the functions that transform RGB or XYZ directly into LMS and color opponents.

## CAM16

See [cam16](cam16) for implementation, and [Li, Li, Wang et al, 2017](#references) for description.

## CIELAB

L\*a\*b\* is defined in the CIELAB color space: https://en.wikipedia.org/wiki/CIELAB_color_space

# HCT

See [hct](hct) for implementation.

[material design](https://material.io/blog/science-of-color-design) uses components of CAM16 and LAB to create an HCT (hue, chroma, tone) space that works well for automatically creating different color shades for GUIs.  The implementation in a variety of languages (excluding Go) is on github at: https://github.com/material-foundation/material-color-utilities -- we leveraged this code for our HCT implementation in Go.

* Hue is CAM16.Hue
* Chroma is CAM16.Chroma
* Tone is LAB Lstar (L\*)

![hct colorspace](examples/hctspace/hctspace.png)

# Color in V1

Two effective populations of cells in V1: double-opponent and single-opponent

* Double-opponent are most common, and define an edge in color space (e.g., R-G edge) by having offset opposing lobes of a gabor (e.g., one lobe is R+G- and the other lobe is G+R-) – this gives the usual zero response for uniform illumination, but a nice contrast response. We should probably turn on color responses in general in our V1 pathway, esp if it is just RG and BY instead of all those other guys. Can also have the color just be summarized in the PI polarity independent pathway.

* Single-opponent which are similar-sized gaussians with opponent R-G and B-Y tuning. These are much fewer, and more concentrated in these CO-blob regions, that go to the “thin” V2 stripes. But the divisions are not perfect..

# References

* Conway, 2001: double-opponent color sensitivity – respond to spatial changes in color, not just raw color contrast – this is key for color constancy and making the color pathway much more efficient – a single-opponent dynamic causes entire color regions to be activated, instead of just activating for changes in color, which is the key point about efficient retinal coding in the luminance domain – just code for local changes, not broad regions. BUT this type of cell is not typically found and other mechanisms exist..

* Gegenfurtner, 2003: nature neuroscience review of color highly recommended – lots of key excerpts on page

* Hellwig, L., & Fairchild, M. D. (2022). Brightness, lightness, colorfulness, and chroma in CIECAM02 and CAM16. Color Research & Application, 47(5), 1083–1095. https://doi.org/10.1002/col.22792

* Solomon & Lennie, 2007: lower-level paper with some nice diagrams and generally consistent conclusions.

* Moroney, N., Fairchild, M. D., Hunt, R. W. G., Li, C., Luo, M. R., & Newman, T. (2002). The CIECAM02 Color Appearance Model. Color and Imaging Conference, 2002(1), 23–27.

* Field et al., 2010: recording in the retina – wow! but not sure of implications.

* Shapley & Hawken, 2011: review – lots of good stuff in here and some strong key conclusions

* Zhang et al., 2012: implements single and double opponent mechs in various models and shows that they work well! – though they do add a R-Cyan channel, and don’t seem to actually use the single opponent channel?? not too clear about that..

* Yang et al., 2013: uses SO and DO but not sure again about SO usage.. maybe just along way to DO.


