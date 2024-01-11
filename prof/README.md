# prof

Provides very basic but effective profiling of targeted functions or code sections, which can often be more informative than generic cpu profiling.

Here's how you use it:

```Go
  // somewhere near start of program (e.g., using flag package)
  profFlag := flag.Bool("prof", false, "turn on targeted profiling")
  ...
  flag.Parse()
  prof.Profiling = *profFlag
  ...
  // surrounding the code of interest:
  pr := prof.Start("name of function")
  ... code
  pr.End()
  ...
  // at end or whenever you've got enough data:
  prof.Report(time.Millisecond) // or time.Second or whatever
``` 

 
