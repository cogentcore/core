#!/bin/bash

# this regenerates the ls-R file in texmf, which is needed whenever files are added or removed

cd texmf
texhash .

