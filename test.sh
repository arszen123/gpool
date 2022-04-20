#!/usr/bin/sh

# Run test in all sub-modules
find . -type f -name '*.go' -execdir go test ./ \;