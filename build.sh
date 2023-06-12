#! /bin/bash

rm -rf ./build/
mkdir ./build/

export CGO_ENABLED=0

GOOS=windows GOARCH=amd64 go build -o ./build/blazehttp_windows.exe   ./cmd/blazehttp
GOOS=darwin  GOARCH=arm64 go build -o ./build/blazehttp_mac_m1        ./cmd/blazehttp 
GOOS=darwin  GOARCH=amd64 go build -o ./build/blazehttp_mac_x64       ./cmd/blazehttp 
GOOS=linux   GOARCH=arm64 go build -o ./build/blazehttp_linux_arm64   ./cmd/blazehttp 
GOOS=linux   GOARCH=amd64 go build -o ./build/blazehttp_linux_x64     ./cmd/blazehttp 
