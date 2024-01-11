# grows

Package grows provides "boilerplate" wrapper functions for the Go standard io functions to Read, Open, Write, and Save, with implementations for commonly used encoding formats.

The top-level `grows` functions define standard `Encoder` and `Decoder` interfaces, and functions to return these. 

The specific encoder format implementations provide these `EncoderFunc` and `DecoderFunc` args.

Buffered io is used, and errors are returned.

