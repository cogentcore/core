The color axes in the human brain are R-G and B-Y. In color speak L = R, M = G, S = B (long, med, short wavelength) –- it is very clear that this is how color is coded in the LGN thalamus.

# Color spaces and computational color

i.e., how to quickly approximate color vision..

* using float32 components throughout

* Here’s what is implemented: https://en.wikipedia.org/wiki/CIECAM02 – transforms RGB -> XYZ -> LMS -> color opponents which are the L, a, b values used in CIECAM0

* paper for this: Moroney et al., 2002

* Color spaces based on LMS cones: https://en.wikipedia.org/wiki/LMS_color_space

* Standard CIE 1931 XYZ color space: https://en.wikipedia.org/wiki/CIE_1931_color_space

* Standard color checker: https://en.wikipedia.org/wiki/ColorChecker

# Color in V1

Two effective populations of cells in V1: double-opponent and single-opponent

* Double-opponent are most common, and define an edge in color space (e.g., R-G edge) by having offset opposing lobes of a gabor (e.g., one lobe is R+G- and the other lobe is G+R-) – this gives the usual zero response for uniform illumination, but a nice contrast response. We should probably turn on color responses in general in our V1 pathway, esp if it is just RG and BY instead of all those other guys. Can also have the color just be summarized in the PI polarity independent pathway.

* Single-opponent which are similar-sized gaussians with opponent R-G and B-Y tuning. These are much fewer, and more concentrated in these CO-blob regions, that go to the “thin” V2 stripes. But the divisions are not perfect..

# Papers

* Conway, 2001: double-opponent color sensitivity – respond to spatial changes in color, not just raw color contrast – this is key for color constancy and making the color pathway much more efficient – a single-opponent dynamic causes entire color regions to be activated, instead of just activating for changes in color, which is the key point about efficient retinal coding in the luminance domain – just code for local changes, not broad regions. BUT this type of cell is not typically found and other mechanisms exist..

* Gegenfurtner, 2003: nature neuroscience review of color highly recommended – lots of key excerpts on page

* Solomon & Lennie, 2007: lower-level paper with some nice diagrams and generally consistent conclusions.

* Field et al., 2010: recording in the retina – wow! but not sure of implications.

* Shapley & Hawken, 2011: review – lots of good stuff in here and some strong key conclusions

* Zhang et al., 2012: implements single and double opponent mechs in various models and shows that they work well! – though they do add a R-Cyan channel, and don’t seem to actually use the single opponent channel?? not too clear about that..

* Yang et al., 2013: uses SO and DO but not sure again about SO usage.. maybe just along way to DO.


