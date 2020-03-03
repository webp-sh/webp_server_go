#!/usr/bin/env bash
cd ..
git pull
platform=$(uname -a)

if [[ $platform =~ "Darwin" ]]
then
    go build -o webp-server-darwin-amd64 webp-server.go
elif [[ $platform =~ "x86_64" ]];then
    go build -o webp-server-unix-amd64 webp-server.go
else
    go build -o webp-server-linux-amd64 webp-server.go
fi
