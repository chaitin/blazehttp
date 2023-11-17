#! /bin/bash

rm -rf ./build/
mkdir ./build/

go build -o ./build/blazehttp ./cmd/blazehttp
