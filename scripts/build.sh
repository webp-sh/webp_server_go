#!/bin/bash

CGO_ENABLED=0

if [ "${1}" == "windows" ]
then
    go build -v -ldflags "-s -w" -o builds/webp-server-${1}-${2}.exe
elif [ "${1}" == "osx" ]
then
    go build -v -ldflags "-s -w" -o builds/webp-server-darwin-${2}
else
    go build -v -ldflags "-s -w" -o builds/webp-server-${1}-${2}
fi

for file in builds/*
do
    sha256sum ${file} > builds/${file}.sha256
done

echo "build done!"
ls builds