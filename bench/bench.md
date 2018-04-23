# Benchmarks for GoGi

`Control+Alt+F` is full render, and `Control+Alt+G` is Re Render

**VERY IMPORTANT** must run benchmarks from a go build and NOT from dlv debug session.

## 2018 - 04 - 22

After converting everything to float32, but before optimizing Styling, measure
string, other render things.  Main conclusions:
* fill is the slow thing about rendering, not stroke.  don't even worry about stroke
* unlike before the publish / copy are very fast..
* measurestring seems to be happening way too much, especially on re-render, where it shouldn't happen at all!
* Style is also way too slow -- working on that by caching everything.

### widgets.go main

```
Starting BenchmarkFullRender
Time for 50 Re-Renders:         4.42 s
     Node2D.Render2DTree:	Tot:	     3592.02	Avg:	       71.84	N:	    50	Pct:	44.36
              Paint.fill:	Tot:	     3055.90	Avg:	        1.80	N:	  1700	Pct:	37.74
      Node2D.Style2DTree:	Tot:	      477.71	Avg:	        9.55	N:	    50	Pct:	 5.90
     Paint.MeasureString:	Tot:	      298.26	Avg:	        0.25	N:	  1200	Pct:	 3.68
        Paint.drawString:	Tot:	      186.04	Avg:	        0.37	N:	   500	Pct:	 2.30
     Node2D.Layout2DTree:	Tot:	      177.39	Avg:	        1.77	N:	   100	Pct:	 2.19
       Node2D.Size2DTree:	Tot:	      145.35	Avg:	        2.91	N:	    50	Pct:	 1.80
            Paint.stroke:	Tot:	       90.27	Avg:	        0.07	N:	  1250	Pct:	 1.11
          win.FullUpdate:	Tot:	       30.67	Avg:	        0.61	N:	    50	Pct:	 0.38
     win.Publish.Publish:	Tot:	       24.64	Avg:	        0.49	N:	    50	Pct:	 0.30
       Node2D.Init2DTree:	Tot:	       12.71	Avg:	        0.25	N:	    50	Pct:	 0.16
        win.Publish.Copy:	Tot:	        5.83	Avg:	        0.12	N:	    50	Pct:	 0.07
```

```
Starting BenchmarkReRender
Time for 50 Re-Renders:         3.56 s
     Node2D.Render2DTree:	Tot:	     3557.91	Avg:	       71.16	N:	    50	Pct:	50.28
              Paint.fill:	Tot:	     3027.33	Avg:	        1.78	N:	  1700	Pct:	42.78
        Paint.drawString:	Tot:	      183.71	Avg:	        0.37	N:	   500	Pct:	 2.60
     Paint.MeasureString:	Tot:	      149.78	Avg:	        0.21	N:	   700	Pct:	 2.12
            Paint.stroke:	Tot:	       85.95	Avg:	        0.07	N:	  1250	Pct:	 1.21
     win.Publish.Publish:	Tot:	       32.37	Avg:	        0.65	N:	    50	Pct:	 0.46
          win.FullUpdate:	Tot:	       30.43	Avg:	        0.61	N:	    50	Pct:	 0.43
        win.Publish.Copy:	Tot:	        6.12	Avg:	        0.12	N:	    50	Pct:	 0.09
     Node2D.Layout2DTree:	Tot:	        2.90	Avg:	        0.06	N:	    50	Pct:	 0.04
```

Setting vp.Fill = false in widgets.go -- those 50 full vp fills were
accounting for about a second, and in general we need a much better optimized
filler for monochrome.

```
Time for 50 Re-Renders:         2.50 s
     Node2D.Render2DTree:	Tot:	     2500.99	Avg:	       50.02	N:	    50	Pct:	50.38
              Paint.fill:	Tot:	     1994.59	Avg:	        1.21	N:	  1650	Pct:	40.18
```

Using optimized WinTex.Fill for base VP and For Frame cuts that down significantly:

```
Time for 50 Re-Renders:         1.64 s
     Node2D.Render2DTree:	Tot:	     1641.51	Avg:	       32.83	N:	    50	Pct:	51.02
              Paint.fill:	Tot:	     1101.34	Avg:	        0.69	N:	  1600	Pct:	34.23
```

And replacing Widget and Icon fills with optimized image.Uniform works even better, cutting down number of fills needed:

``` Go
draw.Draw(vp.Pixels, vp.Pixels.Bounds(), &image.Uniform{&vp.Style.Background.Color}, image.ZP, draw.Src)
```

```
Time for 50 Re-Renders:         1.58 s
     Node2D.Render2DTree:	Tot:	     1579.49	Avg:	       31.59	N:	    50	Pct:	51.40
              Paint.fill:	Tot:	     1020.17	Avg:	        1.46	N:	   700	Pct:	33.20
```

And for GoGiEditor, even more impressive:

```
Starting BenchmarkReRender
Time for 50 Re-Renders:         6.95 s
     Node2D.Render2DTree:	Tot:	     6946.76	Avg:	      138.94	N:	    50	Pct:	51.19
              Paint.fill:	Tot:	     3435.96	Avg:	        1.56	N:	  2200	Pct:	25.32
```



### GoGi Editor on widgets.go

```
Starting BenchmarkFullRender
Starting Targeted Profiling
Time for 50 Re-Renders:        28.85 s
     Node2D.Render2DTree:	Tot:	    11493.60	Avg:	      229.87	N:	    50	Pct:	26.60
      Node2D.Style2DTree:	Tot:	     9490.44	Avg:	      189.81	N:	    50	Pct:	21.97
              Paint.fill:	Tot:	     8118.12	Avg:	        0.71	N:	 11450	Pct:	18.79
     Paint.MeasureString:	Tot:	     5090.84	Avg:	        0.24	N:	 21600	Pct:	11.78
     Node2D.Layout2DTree:	Tot:	     3919.84	Avg:	       39.20	N:	   100	Pct:	 9.07
       Node2D.Size2DTree:	Tot:	     3277.29	Avg:	       65.55	N:	    50	Pct:	 7.59
        Paint.drawString:	Tot:	      959.90	Avg:	        0.13	N:	  7150	Pct:	 2.22
       Node2D.Init2DTree:	Tot:	      547.22	Avg:	       10.94	N:	    50	Pct:	 1.27
            Paint.stroke:	Tot:	      218.55	Avg:	        0.06	N:	  3750	Pct:	 0.51
          win.FullUpdate:	Tot:	       56.97	Avg:	        1.14	N:	    50	Pct:	 0.13
     win.Publish.Publish:	Tot:	       22.28	Avg:	        0.45	N:	    50	Pct:	 0.05
        win.Publish.Copy:	Tot:	        6.19	Avg:	        0.12	N:	    50	Pct:	 0.01
```
		
```
Starting BenchmarkReRender
Starting Targeted Profiling
Time for 50 Re-Renders:        11.75 s
     Node2D.Render2DTree:	Tot:	    11746.82	Avg:	      234.94	N:	    50	Pct:	50.46
              Paint.fill:	Tot:	     8317.83	Avg:	        0.73	N:	 11450	Pct:	35.73
     Paint.MeasureString:	Tot:	     1928.51	Avg:	        0.21	N:	  9350	Pct:	 8.28
        Paint.drawString:	Tot:	      982.89	Avg:	        0.14	N:	  7150	Pct:	 4.22
            Paint.stroke:	Tot:	      218.66	Avg:	        0.06	N:	  3750	Pct:	 0.94
          win.FullUpdate:	Tot:	       55.94	Avg:	        1.12	N:	    50	Pct:	 0.24
     win.Publish.Publish:	Tot:	       19.61	Avg:	        0.39	N:	    50	Pct:	 0.08
        win.Publish.Copy:	Tot:	        6.40	Avg:	        0.13	N:	    50	Pct:	 0.03
     Node2D.Layout2DTree:	Tot:	        2.65	Avg:	        0.05	N:	    50	Pct:	 0.01
```

After fixing fills and optimizing the style, down to 1/2 the time for full render:

```
Starting BenchmarkFullRender
Starting Targeted Profiling
Time for 50 Re-Renders:        14.61 s
     Node2D.Render2DTree:	Tot:	     7386.06	Avg:	      147.72	N:	    50	Pct:	29.50
     Paint.MeasureString:	Tot:	     5585.40	Avg:	        0.26	N:	 21600	Pct:	22.31
       Node2D.Size2DTree:	Tot:	     3620.81	Avg:	       72.42	N:	    50	Pct:	14.46
              Paint.fill:	Tot:	     3610.36	Avg:	        1.64	N:	  2200	Pct:	14.42
      Node2D.Style2DTree:	Tot:	     1663.35	Avg:	       33.27	N:	    50	Pct:	 6.64
     Node2D.Layout2DTree:	Tot:	     1194.25	Avg:	       11.94	N:	   100	Pct:	 4.77
        Paint.drawString:	Tot:	     1039.29	Avg:	        0.15	N:	  7150	Pct:	 4.15
       Node2D.Init2DTree:	Tot:	      612.39	Avg:	       12.25	N:	    50	Pct:	 2.45
            Paint.stroke:	Tot:	      232.19	Avg:	        0.06	N:	  3750	Pct:	 0.93
          win.FullUpdate:	Tot:	       57.50	Avg:	        1.15	N:	    50	Pct:	 0.23
     win.Publish.Publish:	Tot:	       26.28	Avg:	        0.53	N:	    50	Pct:	 0.10
        win.Publish.Copy:	Tot:	        6.34	Avg:	        0.13	N:	    50	Pct:	 0.03
```
