# scanx

Warning: Scanx is beta software. There are known bugs being worked on. Please use scanFT or scanGV unless you want to test each svg file your application may use.

Scanx is a fast antialiaser supporting the draw.Image interface and image.RGBA and xgraphics.Image types in particular. It is intended for use with the rasterx package.

Scanx replaces the Painter interface with the Spanner interface that allows for more direct writing to an underlying image type. Scanx has two types that satisfy the Spanner interface; ImgSpanner and LinkListSpanner.

ImgSpanner draw into any image that supports the draw.Image interface. It is optimized for image.RGBA and xgraphics.Image types.

LinkListSpanner supports the same Image types as ImgSpanner, but stores the spans in y linked lists, where y is the height of the image. It is faster than ImgSpanner for svg icons where the paths overlap significantly, since it only writes to the image after all the spans are collected. The increase in speed is particually significant when drawing to a large image, like a high resolution monitor. However, LinkListSpanner does not support gradients, so if you are using them, you should use ImgSpanner instead.

# Example using ImgSpanner:
```golang
bounds     = image.Rect(0, 0, w, h)
img        = image.NewRGBA(bounds)
spanner     = scanx.NewImgSpanner(img)
scanner    = scanx.NewScanner(spanner, w, h)
raster = rasterx.NewDasher(w, h, scanner)
//Use the raster to draw and the results go to the img
``` 
# Example using LinkListSpanner:
```golang  
bounds     = image.Rect(0, 0, w, h)
img        = image.NewRGBA(bounds)
spanner    = &scanx.LinkListSpanner{}
spanner.SetBounds(bounds)
scanner    = scanx.NewScanner(spanner, w, h)
raster = rasterx.NewDasher(w, h, scanner)
//Use the raster to draw ..
//This draws the accumulated spans onto the image
spanner.DrawToImage(img)
//Get the spanner ready for another image
spanner.Clear()
``` 
# Test results in comparison to scanFT and scanGV
Images for the svg files in the test folder have all been generated and compared pixel for pixel using ScanFT, ImgSpanner and LinkListSpanner. ImgSpanner and LinkListSpanner generated images are all identical except in the case of gradients, which LinkListSpanner does not at this time support. ScanFT will differ from ImgScanner and LinkList spanner in some pixel values, usually by one digit, but in cases with multiple semitransparent overlays the effect can be cummulative. The highest difference in the data set is found in the randspot.svg file, where for some pixels the total difference is 4, although it is hard to see any difference visually.

Below are benchmark results using files in test/lanscapeIcons and the indicated spanner or scanner. They are draw at 0.5, 1, 5, and 15 times native resolution.

```
goos: linux
goarch: amd64
pkg: github.com/srwiley/scanx

Resolution: 0.5x 
BenchmarkLinkListSpanner5-16      	     100	  15692425 ns/op	   37215 B/op	     634 allocs/op
BenchmarkImgSpanner5-16           	     100	  14352161 ns/op	    8708 B/op	     634 allocs/op
BenchmarkFTScanner5-16            	      50	  24019383 ns/op	    3699 B/op	     321 allocs/op
BenchmarkGVScanner5-16            	       3	 362814157 ns/op	    3045 B/op	     321 allocs/op

Resolution: 1x
BenchmarkLinkListSpanner10-16     	      50	  31774480 ns/op	  128512 B/op	     634 allocs/op
BenchmarkImgSpanner10-16          	      50	  35224671 ns/op	   12805 B/op	     634 allocs/op
BenchmarkFTScanner10-16           	      20	  75619298 ns/op	   11241 B/op	     321 allocs/op
BenchmarkGVScanner10-16           	       1	1431206955 ns/op	    3056 B/op	     321 allocs/op

Resolution: 5x
BenchmarkLinkListSpanner50-16     	       5	 227859937 ns/op	 6047212 B/op	     642 allocs/op
BenchmarkImgSpanner50-16          	       2	 513577800 ns/op	  888680 B/op	     644 allocs/op
BenchmarkFTScanner50-16           	       1	1644670841 ns/op	  691184 B/op	     324 allocs/op
BenchmarkGVScanner50-16           	       1	38052841313 ns/op	26217456 B/op	     322 allocs/op

Resolution: 15x
BenchmarkLinkListSpanner150-16    	       1	1344679006 ns/op	93257568 B/op	     679 allocs/op
BenchmarkImgSpanner150-16         	       1	4173881468 ns/op	 5826384 B/op	     665 allocs/op
BenchmarkFTScanner150-16          	       1	10811718931 ns/op	 2788336 B/op	     325 allocs/op
BenchmarkGVScanner150-16          	       1	250690083743 ns/op	235934720 B/op	     327 allocs/op
```
The results indicate the ImgScanner is consistently faster than scanFT or scanGV. Also LinkListSpanner usually does better with this data set as size of the graphic increases. Also note that some svg files can perform quite badly using the LinkListSpanner, such as rl.svg in the testdata/svg folder. This file consists of lots of random lines that slow the list generation.
