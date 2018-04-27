# Benchmarks for GoGi

`Control+Alt+F` is full render, and `Control+Alt+G` is Re Render

**VERY IMPORTANT** must run benchmarks from a go build and NOT from dlv debug session.

* https://github.com/google/pprof/blob/master/doc/pprof.md
	+ go tool pprof cpu.prof 
	+ list Style2D to see all the stuff happening in Style2D

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

### widgets.go main

```
Starting BenchmarkFullRender
Time for 50 Re-Renders:         1.59 s
     Node2D.Render2DTree:	Tot:	     1398.89	Avg:	       27.98	N:	    50	Pct:	49.76
              Paint.fill:	Tot:	      988.34	Avg:	        1.41	N:	   700	Pct:	35.15
        Paint.drawString:	Tot:	      142.94	Avg:	        0.29	N:	   500	Pct:	 5.08
            Paint.stroke:	Tot:	       83.11	Avg:	        0.07	N:	  1250	Pct:	 2.96
      Node2D.Style2DTree:	Tot:	       71.01	Avg:	        1.42	N:	    50	Pct:	 2.53
     Node2D.Layout2DTree:	Tot:	       43.10	Avg:	        0.43	N:	   100	Pct:	 1.53
          win.FullUpdate:	Tot:	       27.62	Avg:	        0.55	N:	    50	Pct:	 0.98
     win.Publish.Publish:	Tot:	       21.92	Avg:	        0.44	N:	    50	Pct:	 0.78
       Node2D.Init2DTree:	Tot:	       12.20	Avg:	        0.24	N:	    50	Pct:	 0.43
     Paint.MeasureString:	Tot:	       10.85	Avg:	        0.01	N:	   900	Pct:	 0.39
       Node2D.Size2DTree:	Tot:	        5.84	Avg:	        0.12	N:	    50	Pct:	 0.21
        win.Publish.Copy:	Tot:	        5.68	Avg:	        0.11	N:	    50	Pct:	 0.20
```

```
Starting BenchmarkReRender
Time for 50 Re-Renders:         1.37 s
     Node2D.Render2DTree:	Tot:	     1373.74	Avg:	       27.47	N:	    50	Pct:	52.41
              Paint.fill:	Tot:	      965.07	Avg:	        1.38	N:	   700	Pct:	36.82
        Paint.drawString:	Tot:	      138.89	Avg:	        0.28	N:	   500	Pct:	 5.30
            Paint.stroke:	Tot:	       81.46	Avg:	        0.07	N:	  1250	Pct:	 3.11
          win.FullUpdate:	Tot:	       26.86	Avg:	        0.54	N:	    50	Pct:	 1.02
     win.Publish.Publish:	Tot:	       22.63	Avg:	        0.45	N:	    50	Pct:	 0.86
     Paint.MeasureString:	Tot:	        5.92	Avg:	        0.01	N:	   500	Pct:	 0.23
        win.Publish.Copy:	Tot:	        5.63	Avg:	        0.11	N:	    50	Pct:	 0.21
     Node2D.Layout2DTree:	Tot:	        0.83	Avg:	        0.02	N:	    50	Pct:	 0.03
```

### GoGi Editor on widgets.go

(at start of benchmarking, full render total was 28s, and re-render was 12s -- major factors of improvement)

```
Starting BenchmarkFullRender
Time for 50 Re-Renders:         7.72 s
     Node2D.Render2DTree:	Tot:	     4569.74	Avg:	       91.39	N:	    50	Pct:	38.12
              Paint.fill:	Tot:	     3279.54	Avg:	        1.49	N:	  2200	Pct:	27.36
      Node2D.Style2DTree:	Tot:	     1466.13	Avg:	       29.32	N:	    50	Pct:	12.23
     Node2D.Layout2DTree:	Tot:	     1007.39	Avg:	       10.07	N:	   100	Pct:	 8.40
        Paint.drawString:	Tot:	      724.50	Avg:	        0.10	N:	  7150	Pct:	 6.04
       Node2D.Init2DTree:	Tot:	      516.63	Avg:	       10.33	N:	    50	Pct:	 4.31
            Paint.stroke:	Tot:	      202.36	Avg:	        0.05	N:	  3800	Pct:	 1.69
       Node2D.Size2DTree:	Tot:	       79.57	Avg:	        1.59	N:	    50	Pct:	 0.66
     Paint.MeasureString:	Tot:	       57.79	Avg:	        0.00	N:	 18400	Pct:	 0.48
          win.FullUpdate:	Tot:	       52.58	Avg:	        1.05	N:	    50	Pct:	 0.44
     win.Publish.Publish:	Tot:	       25.39	Avg:	        0.51	N:	    50	Pct:	 0.21
        win.Publish.Copy:	Tot:	        6.63	Avg:	        0.13	N:	    50	Pct:	 0.06
```

```
Starting BenchmarkReRender
Time for 50 Re-Renders:         4.56 s
     Node2D.Render2DTree:	Tot:	     4558.45	Avg:	       91.17	N:	    50	Pct:	51.38
              Paint.fill:	Tot:	     3277.28	Avg:	        1.49	N:	  2200	Pct:	36.94
        Paint.drawString:	Tot:	      721.19	Avg:	        0.10	N:	  7150	Pct:	 8.13
            Paint.stroke:	Tot:	      202.33	Avg:	        0.05	N:	  3800	Pct:	 2.28
          win.FullUpdate:	Tot:	       52.56	Avg:	        1.05	N:	    50	Pct:	 0.59
     Paint.MeasureString:	Tot:	       27.85	Avg:	        0.00	N:	  7250	Pct:	 0.31
     win.Publish.Publish:	Tot:	       24.86	Avg:	        0.50	N:	    50	Pct:	 0.28
        win.Publish.Copy:	Tot:	        6.54	Avg:	        0.13	N:	    50	Pct:	 0.07
     Node2D.Layout2DTree:	Tot:	        0.69	Avg:	        0.01	N:	    50	Pct:	 0.01
```
