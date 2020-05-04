# Benchmarks for GoGi

`Control+Alt+F` is full render, and `Control+Alt+G` is Re Render

**VERY IMPORTANT** must run benchmarks from a go build and NOT from dlv debug session.

* to get more interpretable results from pprof: `export GOMAXPROCS=1`

* https://github.com/google/pprof/blob/master/doc/pprof.md
	+ pprof cpu.prof 
	+ list Style2D to see all the stuff happening in Style2D
	+ pprof -http=localhost:5555 cpu.prof

## 2019 - 05 - 15 -- bespoke styling functions

This is from the ra25 emergent leabra demo, pulling up the slice of verticies for Hidden2 layer, which is 2880 verticies.  It was horrendously long but then I removed redundant Config calls in Style2D and an extra rebuild during window presentation, and that helped a lot.  But it is still way too slow.

```
       Node2D.Init2DTree:	Tot:	     2156.69	Avg:	      431.34	N:	     5	Pct:	32.84
      Node2D.Style2DTree:	Tot:	     2149.90	Avg:	      429.98	N:	     5	Pct:	32.74
     Node2D.Layout2DTree:	Tot:	      883.03	Avg:	      220.76	N:	     4	Pct:	13.45
       StyleFields.Style:	Tot:	      568.96	Avg:	        0.00	N:	443625	Pct:	 8.66
       Node2D.Size2DTree:	Tot:	      282.20	Avg:	       47.03	N:	     6	Pct:	 4.30
      StyleFields.ToDots:	Tot:	      259.09	Avg:	        0.00	N:	795073	Pct:	 3.95
       TextField.Style2D:	Tot:	      113.57	Avg:	        0.02	N:	  5760	Pct:	 1.73
             win.Publish:	Tot:	       75.45	Avg:	        7.54	N:	    10	Pct:	 1.15
     Node2D.Render2DTree:	Tot:	       34.94	Avg:	        6.99	N:	     5	Pct:	 0.53
        TextRenderLayout:	Tot:	       25.36	Avg:	        0.00	N:	  5770	Pct:	 0.39
              Paint.fill:	Tot:	        8.35	Avg:	        0.03	N:	   241	Pct:	 0.13
       vp.ReRender2DNode:	Tot:	        2.54	Avg:	        0.64	N:	     4	Pct:	 0.04
            Paint.stroke:	Tot:	        2.49	Avg:	        0.01	N:	   307	Pct:	 0.04
  win.UploadAllViewports:	Tot:	        2.34	Avg:	        2.34	N:	     1	Pct:	 0.04
      win.UploadVpRegion:	Tot:	        1.28	Avg:	        0.32	N:	     4	Pct:	 0.02
              RenderText:	Tot:	        0.54	Avg:	        0.01	N:	    64	Pct:	 0.01
```

I mistakenly thought the issue was styling because style2d seems to be taking so long, and I had long been suspicious of the reflect-based code.  but alas, after spending too long fixing that, it had relatively minor effects:

```
       Node2D.Init2DTree:	Tot:	     2164.90	Avg:	      240.54	N:	     9	Pct:	33.43
      Node2D.Style2DTree:	Tot:	     2011.75	Avg:	      335.29	N:	     6	Pct:	31.07
     Node2D.Layout2DTree:	Tot:	      934.22	Avg:	      133.46	N:	     7	Pct:	14.43
          StyleFromProps:	Tot:	      452.55	Avg:	        0.00	N:	553079	Pct:	 6.99
       Node2D.Size2DTree:	Tot:	      276.71	Avg:	       27.67	N:	    10	Pct:	 4.27
      StyleFields.ToDots:	Tot:	      261.39	Avg:	        0.00	N:	795132	Pct:	 4.04
             win.Publish:	Tot:	      137.09	Avg:	        5.71	N:	    24	Pct:	 2.12
       TextField.Style2D:	Tot:	      107.41	Avg:	        0.02	N:	  5760	Pct:	 1.66
     Node2D.Render2DTree:	Tot:	       41.46	Avg:	        4.61	N:	     9	Pct:	 0.64
       StyleFields.Style:	Tot:	       35.65	Avg:	        0.00	N:	103717	Pct:	 0.55
        TextRenderLayout:	Tot:	       24.20	Avg:	        0.00	N:	  5776	Pct:	 0.37
              Paint.fill:	Tot:	        9.47	Avg:	        0.04	N:	   258	Pct:	 0.15
  win.UploadAllViewports:	Tot:	        8.85	Avg:	        4.42	N:	     2	Pct:	 0.14
       vp.ReRender2DNode:	Tot:	        3.82	Avg:	        0.64	N:	     6	Pct:	 0.06
            Paint.stroke:	Tot:	        3.38	Avg:	        0.01	N:	   343	Pct:	 0.05
      win.UploadVpRegion:	Tot:	        1.85	Avg:	        0.31	N:	     6	Pct:	 0.03
              RenderText:	Tot:	        0.78	Avg:	        0.01	N:	    77	Pct:	 0.01
            win.UploadVp:	Tot:	        0.30	Avg:	        0.30	N:	     1	Pct:	 0.00
```

A massive revisiting of all of the Ki code resulted in major speedups of tree traversal and re-recognition that the *Fast* versions of the AddNewChild methods are *essential* for doing things efficiently.  Actually the Config version is much better already -- but sliceview is not using that.
Anyway, many orders of magnitude speedup are now at hand.  Details here:

https://docs.google.com/spreadsheets/d/1WWorjrVao6sLRuwKguyRnnuY51DxsZOpokZHzGtmnbE/edit?usp=sharing

And yet, all of this has not materially affected the actual perf -- it is NOT the 
uniquify names that was killing it here.  Init is cut in half though so that's good.
Something remains in style that we need to find..

```
      Node2D.Style2DTree:	Tot:	     2135.90	Avg:	      177.99	N:	    12	Pct:	40.08
       Node2D.Init2DTree:	Tot:	     1155.99	Avg:	       96.33	N:	    12	Pct:	21.69
     Node2D.Layout2DTree:	Tot:	      793.05	Avg:	       88.12	N:	     9	Pct:	14.88
          StyleFromProps:	Tot:	      488.79	Avg:	        0.00	N:	553092	Pct:	 9.17
      StyleFields.ToDots:	Tot:	      258.28	Avg:	        0.00	N:	795138	Pct:	 4.85
       Node2D.Size2DTree:	Tot:	      182.03	Avg:	       14.00	N:	    13	Pct:	 3.42
       TextField.Style2D:	Tot:	      113.92	Avg:	        0.02	N:	  5760	Pct:	 2.14
             win.Publish:	Tot:	       75.81	Avg:	        3.99	N:	    19	Pct:	 1.42
     Node2D.Render2DTree:	Tot:	       38.23	Avg:	        3.48	N:	    11	Pct:	 0.72
       StyleFields.Style:	Tot:	       37.07	Avg:	        0.00	N:	103723	Pct:	 0.70
        TextRenderLayout:	Tot:	       26.06	Avg:	        0.00	N:	  5775	Pct:	 0.49
   ki.Node.UniquifyNames:	Tot:	        7.67	Avg:	        0.00	N:	 25936	Pct:	 0.14
              Paint.fill:	Tot:	        7.48	Avg:	        0.03	N:	   248	Pct:	 0.14
            Paint.stroke:	Tot:	        2.48	Avg:	        0.01	N:	   318	Pct:	 0.05
       vp.ReRender2DNode:	Tot:	        2.13	Avg:	        0.27	N:	     8	Pct:	 0.04
  win.UploadAllViewports:	Tot:	        2.04	Avg:	        2.04	N:	     1	Pct:	 0.04
      win.UploadVpRegion:	Tot:	        1.60	Avg:	        0.20	N:	     8	Pct:	 0.03
              RenderText:	Tot:	        0.61	Avg:	        0.01	N:	    66	Pct:	 0.01
            win.UploadVp:	Tot:	        0.56	Avg:	        0.28	N:	     2	Pct:	 0.01
```

            
## 2018 - 05 - 29 -- switch to rasterx



## 2018 - 04 - 24

* style is now much faster by just looking at props and compiling all the fields into maps

* LoadFont was happening all the time due to a stupid error -- this was causing MeasureString and everything else font related to happen every time -- extremely slow!

* TextField was calling measurestring all the time and in general had some crazy code -- added a MeasureChars method that gets all the char positions in one slice, and we use that for everything, in much better scrolling etc code.

* Overall performance is now quite acceptable and dominated by Fill still -- can come back to that later.

## 2018 - 04 - 22

After converting everything to float32, but before optimizing Styling, measure
string, other render things.  Main conclusions:
* fill is the slow thing about rendering, not stroke.  don't even worry about stroke
* unlike before the publish / copy are very fast..
* measurestring seems to be happening way too much, especially on re-render, where it shouldn't happen at all!
* Style is also way too slow -- working on that by caching everything.


## Current Benchmark Results

(git history can track prior results.. just keep the current reference results in here, plus perhaps some key transition points)

### GoGi Editor on widgets.go

* now using srwiley/rasterx, based on freetype rasterizer -- fill is over 2x
  faster, and stroke is also faster -- in addition there is only one path so overall memory etc will be faster

* at start of benchmarking, full render total was 28s, and re-render was 12s -- major factors of improvement

* Significant speedup in full re-render by doing a manual version of Inherits, which makes a lot of
sense anyway (doesn't require anything fancy -- just copy) and was mysteriously taking a TON of time.

```
Starting Targeted Profiling, window has 2098 nodes
Time for 50 Re-Renders:         2.62 s
     Node2D.Render2DTree:	Tot:	     1324.12	Avg:	       26.48	N:	    50	Pct:	32.54
              Paint.fill:	Tot:	      877.90	Avg:	        0.19	N:	  4650	Pct:	21.57
      Node2D.Style2DTree:	Tot:	      662.82	Avg:	       13.26	N:	    50	Pct:	16.29
     Node2D.Layout2DTree:	Tot:	      287.18	Avg:	        2.87	N:	   100	Pct:	 7.06
       Node2D.Init2DTree:	Tot:	      192.99	Avg:	        3.86	N:	    50	Pct:	 4.74
       StyleFields.Style:	Tot:	      154.93	Avg:	        0.00	N:	210650	Pct:	 3.81
            Paint.stroke:	Tot:	      149.53	Avg:	        0.01	N:	 12550	Pct:	 3.67
      StyleFields.ToDots:	Tot:	      115.16	Avg:	        0.00	N:	433650	Pct:	 2.83
        TextRenderLayout:	Tot:	       94.39	Avg:	        0.00	N:	 21950	Pct:	 2.32
       Node2D.Size2DTree:	Tot:	       79.23	Avg:	        1.58	N:	    50	Pct:	 1.95
     win.Publish.Publish:	Tot:	       58.19	Avg:	        1.16	N:	    50	Pct:	 1.43
              RenderText:	Tot:	       34.78	Avg:	        0.01	N:	  5850	Pct:	 0.85
  win.UploadAllViewports:	Tot:	       30.32	Avg:	        0.61	N:	    50	Pct:	 0.74
        win.Publish.Copy:	Tot:	        7.08	Avg:	        0.14	N:	    50	Pct:	 0.17
     Style.FromProps.Int:	Tot:	        1.06	Avg:	        0.00	N:	  3350	Pct:	 0.03
```

* Also big speedup on FileView of svn_docs/figs, which was at 37.97 s before:

```
Starting Targeted Profiling, window has 26689 nodes
Time for 50 Re-Renders:        25.31 s
      Node2D.Style2DTree:	Tot:	    11570.93	Avg:	      231.42	N:	    50	Pct:	35.28
       Node2D.Size2DTree:	Tot:	     6565.28	Avg:	      131.31	N:	    50	Pct:	20.02
       StyleFields.Style:	Tot:	     4341.58	Avg:	        0.00	N:	5994450	Pct:	13.24
     Node2D.Layout2DTree:	Tot:	     2899.77	Avg:	       58.00	N:	    50	Pct:	 8.84
      StyleFields.ToDots:	Tot:	     2795.35	Avg:	        0.00	N:	9410950	Pct:	 8.52
     Node2D.Render2DTree:	Tot:	     1882.18	Avg:	       37.64	N:	    50	Pct:	 5.74
       Node2D.Init2DTree:	Tot:	     1398.54	Avg:	       27.97	N:	    50	Pct:	 4.26
              Paint.fill:	Tot:	      871.58	Avg:	        0.83	N:	  1050	Pct:	 2.66
            Paint.stroke:	Tot:	      222.89	Avg:	        0.02	N:	  9350	Pct:	 0.68
              RenderText:	Tot:	       99.05	Avg:	        0.02	N:	  6350	Pct:	 0.30
     win.Publish.Publish:	Tot:	       96.01	Avg:	        1.92	N:	    50	Pct:	 0.29
  win.UploadAllViewports:	Tot:	       41.05	Avg:	        0.82	N:	    50	Pct:	 0.13
        win.Publish.Copy:	Tot:	        7.37	Avg:	        0.15	N:	    50	Pct:	 0.02
        TextRenderLayout:	Tot:	        3.13	Avg:	        0.00	N:	   650	Pct:	 0.01
     Style.FromProps.Int:	Tot:	        0.10	Avg:	        0.00	N:	   100	Pct:	 0.00
```

* Before StyleFields.Inherit optimization:

```
Starting Targeted Profiling, window has 2098 nodes
Time for 50 Re-Renders:         3.66 s
      Node2D.Style2DTree:	Tot:	     1702.94	Avg:	       34.06	N:	    50	Pct:	26.64
     Node2D.Render2DTree:	Tot:	     1254.75	Avg:	       25.09	N:	    50	Pct:	19.63
     StyleFields.Inherit:	Tot:	     1009.91	Avg:	        0.01	N:	104800	Pct:	15.80
              Paint.fill:	Tot:	      817.88	Avg:	        0.18	N:	  4650	Pct:	12.79
     Node2D.Layout2DTree:	Tot:	      363.88	Avg:	        3.64	N:	   100	Pct:	 5.69
      StyleFields.ToDots:	Tot:	      218.69	Avg:	        0.00	N:	433650	Pct:	 3.42
       Node2D.Init2DTree:	Tot:	      184.31	Avg:	        3.69	N:	    50	Pct:	 2.88
       StyleFields.Style:	Tot:	      148.14	Avg:	        0.00	N:	210650	Pct:	 2.32
     Style.FromProps.Int:	Tot:	      147.04	Avg:	        0.00	N:	946550	Pct:	 2.30
            Paint.stroke:	Tot:	      145.89	Avg:	        0.01	N:	 12550	Pct:	 2.28
Style.FromProps.SetRobust:	Tot:	      103.65	Avg:	        0.00	N:	524000	Pct:	 1.62
        TextRenderLayout:	Tot:	       89.17	Avg:	        0.00	N:	 21950	Pct:	 1.39
       Node2D.Size2DTree:	Tot:	       74.50	Avg:	        1.49	N:	    50	Pct:	 1.17
     win.Publish.Publish:	Tot:	       62.95	Avg:	        1.26	N:	    50	Pct:	 0.98
              RenderText:	Tot:	       33.17	Avg:	        0.01	N:	  5850	Pct:	 0.52
  win.UploadAllViewports:	Tot:	       29.04	Avg:	        0.58	N:	    50	Pct:	 0.45
        win.Publish.Copy:	Tot:	        6.77	Avg:	        0.14	N:	    50	Pct:	 0.11
```

* Earlier..

```
Starting BenchmarkFullRender
Starting Std CPU / Mem Profiling
Starting Targeted Profiling, window has 2447 nodes
Time for 50 Re-Renders:         4.28 s
     Node2D.Render2DTree:	Tot:	     2228.51	Avg:	       44.57	N:	    50	Pct:	30.62
              Paint.fill:	Tot:	     1325.77	Avg:	        0.58	N:	  2300	Pct:	18.22
      Node2D.Style2DTree:	Tot:	     1185.75	Avg:	       23.71	N:	    50	Pct:	16.29
     StyleFields.Inherit:	Tot:	      543.99	Avg:	        0.00	N:	135700	Pct:	 7.48
        Paint.drawString:	Tot:	      543.43	Avg:	        0.06	N:	  8500	Pct:	 7.47
       Node2D.Init2DTree:	Tot:	      335.12	Avg:	        6.70	N:	    50	Pct:	 4.60
     Node2D.Layout2DTree:	Tot:	      318.65	Avg:	        3.19	N:	   100	Pct:	 4.38
      StyleFields.ToDots:	Tot:	      294.34	Avg:	        0.00	N:	729600	Pct:	 4.04
            Paint.stroke:	Tot:	      135.76	Avg:	        0.02	N:	  7000	Pct:	 1.87
       Node2D.Size2DTree:	Tot:	      127.64	Avg:	        2.55	N:	    50	Pct:	 1.75
       StyleFields.Style:	Tot:	      106.39	Avg:	        0.00	N:	179300	Pct:	 1.46
     Paint.MeasureString:	Tot:	       73.46	Avg:	        0.00	N:	 22100	Pct:	 1.01
     win.Publish.Publish:	Tot:	       27.49	Avg:	        0.55	N:	    50	Pct:	 0.38
          win.FullUpdate:	Tot:	       25.07	Avg:	        0.50	N:	    50	Pct:	 0.34
        win.Publish.Copy:	Tot:	        6.01	Avg:	        0.12	N:	    50	Pct:	 0.08
```

```
Starting BenchmarkReRender
Starting Targeted Profiling, window has 2447 nodes
Time for 50 Re-Renders:         2.18 s
     Node2D.Render2DTree:	Tot:	     2179.82	Avg:	       43.60	N:	    50	Pct:	51.52
              Paint.fill:	Tot:	     1291.70	Avg:	        0.56	N:	  2300	Pct:	30.53
        Paint.drawString:	Tot:	      532.70	Avg:	        0.06	N:	  8500	Pct:	12.59
            Paint.stroke:	Tot:	      134.13	Avg:	        0.02	N:	  7000	Pct:	 3.17
     Paint.MeasureString:	Tot:	       35.30	Avg:	        0.00	N:	  8500	Pct:	 0.83
     win.Publish.Publish:	Tot:	       26.13	Avg:	        0.52	N:	    50	Pct:	 0.62
          win.FullUpdate:	Tot:	       25.21	Avg:	        0.50	N:	    50	Pct:	 0.60
        win.Publish.Copy:	Tot:	        5.74	Avg:	        0.11	N:	    50	Pct:	 0.14
     Node2D.Layout2DTree:	Tot:	        0.41	Avg:	        0.01	N:	    50	Pct:	 0.01
      StyleFields.ToDots:	Tot:	        0.14	Avg:	        0.00	N:	   100	Pct:	 0.00
```


### GoGi Editor on widgets.go:  OLD bare FreeType renderer

```
Starting BenchmarkFullRender
Starting Targeted Profiling, window has 2447 nodes
Time for 50 Re-Renders:         5.47 s
     Node2D.Render2DTree:	Tot:	     3577.71	Avg:	       71.55	N:	    50	Pct:	36.74
              Paint.fill:	Tot:	     2657.87	Avg:	        1.16	N:	  2300	Pct:	27.29
      Node2D.Style2DTree:	Tot:	     1089.17	Avg:	       21.78	N:	    50	Pct:	11.19
        Paint.drawString:	Tot:	      524.10	Avg:	        0.06	N:	  8500	Pct:	 5.38
     StyleFields.Inherit:	Tot:	      500.25	Avg:	        0.00	N:	135700	Pct:	 5.14
       Node2D.Init2DTree:	Tot:	      304.51	Avg:	        6.09	N:	    50	Pct:	 3.13
     Node2D.Layout2DTree:	Tot:	      293.09	Avg:	        2.93	N:	   100	Pct:	 3.01
      StyleFields.ToDots:	Tot:	      272.50	Avg:	        0.00	N:	729600	Pct:	 2.80
            Paint.stroke:	Tot:	      182.86	Avg:	        0.03	N:	  7000	Pct:	 1.88
       Node2D.Size2DTree:	Tot:	      120.24	Avg:	        2.40	N:	    50	Pct:	 1.23
       StyleFields.Style:	Tot:	       96.65	Avg:	        0.00	N:	179300	Pct:	 0.99
     Paint.MeasureString:	Tot:	       68.24	Avg:	        0.00	N:	 22100	Pct:	 0.70
          win.FullUpdate:	Tot:	       25.30	Avg:	        0.51	N:	    50	Pct:	 0.26
     win.Publish.Publish:	Tot:	       19.03	Avg:	        0.38	N:	    50	Pct:	 0.20
        win.Publish.Copy:	Tot:	        6.07	Avg:	        0.12	N:	    50	Pct:	 0.06
```

```
Starting BenchmarkReRender
Starting Targeted Profiling, window has 2447 nodes
Time for 50 Re-Renders:         3.57 s
     Node2D.Render2DTree:	Tot:	     3566.55	Avg:	       71.33	N:	    50	Pct:	50.92
              Paint.fill:	Tot:	     2653.58	Avg:	        1.15	N:	  2300	Pct:	37.89
        Paint.drawString:	Tot:	      515.26	Avg:	        0.06	N:	  8500	Pct:	 7.36
            Paint.stroke:	Tot:	      181.06	Avg:	        0.03	N:	  7000	Pct:	 2.59
     Paint.MeasureString:	Tot:	       33.33	Avg:	        0.00	N:	  8500	Pct:	 0.48
     win.Publish.Publish:	Tot:	       23.53	Avg:	        0.47	N:	    50	Pct:	 0.34
          win.FullUpdate:	Tot:	       23.46	Avg:	        0.47	N:	    50	Pct:	 0.33
        win.Publish.Copy:	Tot:	        6.81	Avg:	        0.14	N:	    50	Pct:	 0.10
     Node2D.Layout2DTree:	Tot:	        0.39	Avg:	        0.01	N:	    50	Pct:	 0.01
      StyleFields.ToDots:	Tot:	        0.14	Avg:	        0.00	N:	   100	Pct:	 0.00
```

# Ki Benchmarks

Laptop = fast macbook pro as of 5/2020:

Baseline with no locking:

```
oreilly@shadow:~/goki/ki/ki/ > go test -bench .
goos: darwin
goarch: amd64
BenchmarkBuildGuiTree_NodeEmbed-16        	      30	  38502502 ns/op
BenchmarkBuildGuiTree_NodeField-16        	      14	  82690320 ns/op
BenchmarkBuildGuiTree_NodeField2-16       	       7	 159047111 ns/op
BenchmarkBuildGuiTreeSlow_NodeEmbed-16    	       1	9799964786 ns/op
BenchmarkFuncDownMeFirst_NodeEmbed-16     	     264	   4633235 ns/op
BenchmarkFuncDownMeFirst_NodeField-16     	     129	   9027565 ns/op
BenchmarkFuncDownMeFirst_NodeField2-16    	      85	  14319531 ns/op
```

With mutex locking at start / end:

```
BenchmarkBuildGuiTree_NodeEmbed-16        	      30	  39641040 ns/op
BenchmarkBuildGuiTree_NodeField-16        	      14	  85435516 ns/op
BenchmarkBuildGuiTree_NodeField2-16       	       7	 162794630 ns/op
BenchmarkBuildGuiTreeSlow_NodeEmbed-16    	       1	10012321622 ns/op
BenchmarkFuncDownMeFirst_NodeEmbed-16     	     242	   4786027 ns/op
BenchmarkFuncDownMeFirst_NodeField-16     	     123	   9662095 ns/op
BenchmarkFuncDownMeFirst_NodeField2-16    	      78	  15406549 ns/op
```

= roughly 7% slower

With full separate trav state map (fully safe):

```
BenchmarkBuildGuiTree_NodeEmbed-16        	      20	  56353512 ns/op
BenchmarkBuildGuiTree_NodeField-16        	       9	 126170202 ns/op
BenchmarkBuildGuiTree_NodeField2-16       	       5	 216370147 ns/op
BenchmarkBuildGuiTreeSlow_NodeEmbed-16    	       1	9857690075 ns/op
BenchmarkFuncDownMeFirst_NodeEmbed-16     	      64	  19818276 ns/op
BenchmarkFuncDownMeFirst_NodeField-16     	      30	  39991628 ns/op
BenchmarkFuncDownMeFirst_NodeField2-16    	      19	  59633855 ns/op
```

= 400% slower!

Deleting nodes when done -- 2x slower and probably worth it -- this is now the new current impl:

```
BenchmarkBuildGuiTree_NodeEmbed-16        	      20	  50797676 ns/op
BenchmarkBuildGuiTree_NodeField-16        	       9	 119290295 ns/op
BenchmarkBuildGuiTree_NodeField2-16       	       5	 207227280 ns/op
BenchmarkBuildGuiTreeSlow_NodeEmbed-16    	       1	10072070259 ns/op
BenchmarkFuncDownMeFirst_NodeEmbed-16     	     100	  11784102 ns/op
BenchmarkFuncDownMeFirst_NodeField-16     	      54	  22724071 ns/op
BenchmarkFuncDownMeFirst_NodeField2-16    	      32	  36481851 ns/op
```


