#!/bin/bash

mkdir -p bin

ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
  curl -L -o bin/atlas https://release.ariga.io/atlas/atlas-community-linux-amd64-latest
elif [ "$ARCH" = "aarch64" ]; then
  curl -L -o bin/atlas https://release.ariga.io/atlas/atlas-community-linux-arm64-latest
fi

chmod +x bin/atlas
