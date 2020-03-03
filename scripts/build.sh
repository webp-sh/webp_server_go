#!/bin/bash

CGO_ENABLED=0

GOOS=linux
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-linux-amd64

GOOS=linux
GOARCH=arm
go build -x -v -ldflags "-s -w" -o builds/webp-server-linux-arm

GOOS=darwin
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-darwin-amd64

GOOS=windows
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-windows-amd64.exe

echo "build done!"
ls builds