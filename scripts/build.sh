#!/bin/bash

CGO_ENABLED=0

GOOS=linux
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-linux-amd64 ../webp-server.go

GOOS=linux
GOARCH=arm
go build -x -v -ldflags "-s -w" -o builds/webp-server-linux-arm ../webp-server.go

GOOS=darwin
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-darwin-amd64 ../webp-server.go

GOOS=windows
GOARCH=amd64
go build -x -v -ldflags "-s -w" -o builds/webp-server-windows-amd64.exe ../webp-server.go

echo "build done!"
ls builds