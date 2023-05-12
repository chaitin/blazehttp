#! /bin/bash

rm -rf ./build/
mkdir ./build/
GOOS=windows GOARCH=amd64 go build -o ./build/blazehttp_windows.exe   ./cmd/blazehttp
GOOS=darwin  GOARCH=arm64 go build -o ./build/blazehttp_mac_m1.exe    ./cmd/blazehttp 
GOOS=darwin  GOARCH=amd64 go build -o ./build/blazehttp_mac_x86.exe   ./cmd/blazehttp 
GOOS=linux   GOARCH=arm64 go build -o ./build/blazehttp_linux_arm.exe ./cmd/blazehttp 
GOOS=linux   GOARCH=amd64 go build -o ./build/blazehttp_linux_x86.exe ./cmd/blazehttp 
