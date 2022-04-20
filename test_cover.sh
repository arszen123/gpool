#!/usr/bin/bash
dir=$(pwd)/coverage
coverprofiles="$dir/coverprofiles"
mainCoverprofile="$dir/coverage.out"

rm -r $dir
mkdir $dir
mkdir $coverprofiles

# Run coverage on all sub-modules
for file in $(find . -type f -name '*.go' | grep -v "_test.go")
do
  filename=${file/"_test"/""}
  normalizedName=${filename/".go"/""}
  normalizedName=${normalizedName/"./"/""}
  normalizedName=${normalizedName//"/"/"_"}
  directory=${filename%/*}/

  go test ${directory} -coverprofile "$coverprofiles/$normalizedName.out"
done;

# Create a summary of all the coverage profiles
echo "mode: set" > $mainCoverprofile
for file in $(find $coverprofiles -type f -name "*.out")
do
  tail -n +2 -q $file >> $mainCoverprofile
done;

# Open coverage report in browser
go tool cover -html=$mainCoverprofile