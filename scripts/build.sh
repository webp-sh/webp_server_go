#!/bin/bash

CGO_ENABLED=0

go build -x -v -ldflags "-s -w" -o builds/webp-server-linux-${1}

echo "build done!"
ls builds