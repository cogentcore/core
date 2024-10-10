# rand

This example tests the `slrand` random number generation functions.  The `go test` in `slrand` itself tests the Go version of the code against known values, and this example tests the GPU WGSL versions against the Go versions.

The output shows the first 5 sets of random numbers, with the CPU on the first line and the GPU on the second line.  If there are any `*` on any of the lines, then there is a difference between the two, and an error message will be reported at the bottom.

The total time to generate 1 million random numbers is shown at the end, comparing time on the CPU vs. GPU.  On a Mac Book Pro M3 Max laptop, the CPU took roughly _95_ times longer to generate the random numbers than the GPU (and with 10 million, _140 times_).

# Building

There is a `//go:generate` comment directive in `main.go` that calls _the local build_ of `gosl` on the relevant files, so you can do `go generate` followed by `go build` to run it.  You must do `go build` in main `gosl` dir before this will work.

The generated files go into the `shaders/` subdirectory.

Ignore the type alignment checking errors about Uint2 and Vector2 not being an even multiple of 16 bytes -- we have put in the necessary padding.


# Results

```
Running on GPU: Apple M3 Max
Group: 0 Group0
    Role: Storage
        Var: 0:	Counter	Struct	(size: 8)	Values: 1
        Var: 1:	Data	Struct[0xF4240]	(size: 64)	Values: 1

Index	Dif(Ex,Tol)	   CPU   	  then GPU
0	   	U: ff1dae59	6cd10df2	F: 0.86274576	0.37199184	F11: 0.018057441	0.24895307	G: 0.7749991	0.05054265
		U: ff1dae59	6cd10df2	F: 0.86274576	0.37199184	F11: 0.018057441	0.24895307	G: 0.7749991	0.050542645
1	   	U: 936f52f3	5daa6164	F: 0.87315667	0.9577325	F11: 0.7201687	0.93292534	G: 0.82841516	2.2875004
		U: 936f52f3	5daa6164	F: 0.87315667	0.9577325	F11: 0.7201687	0.93292534	G: 0.82841516	2.2875004
2	   	U: 1a9351a6	5109a5a6	F: 0.7390654	0.13888454	F11: 0.87044024	0.66876566	G: -1.4226444	0.024453443
		U: 1a9351a6	5109a5a6	F: 0.7390654	0.13888454	F11: 0.87044024	0.66876566	G: -1.4226445	0.024453443
3	   	U: 19b1b6d2	310630c9	F: 0.8936864	0.29176176	F11: 0.5634876	0.43976986	G: 0.07859113	0.07448565
		U: 19b1b6d2	310630c9	F: 0.8936864	0.29176176	F11: 0.5634876	0.43976986	G: 0.07859113	0.07448566
4	   	U: 41556b7f	eeb8e52c	F: 0.46174365	0.28119206	F11: 0.00079108716	0.87918425	G: 0.2611614	-1.3341242
		U: 41556b7f	eeb8e52c	F: 0.46174365	0.28119206	F11: 0.00079108716	0.87918425	G: 0.26116177	-1.3341241
5	   	U: c034b0a6	7188ed5e	F: 0.9694913	0.6442756	F11: 0.071927555	0.5161969	G: 0.6914865	0.32279235
		U: c034b0a6	7188ed5e	F: 0.9694913	0.6442756	F11: 0.071927555	0.5161969	G: 0.6914865	0.32279235

N: 1000000	 CPU: 104.698542ms	 GPU: 1.095417ms	 Full: 27.570417ms	 CPU/GPU:  95.58
```

