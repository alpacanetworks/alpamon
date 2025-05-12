#!/bin/bash

mkdir -p bin

ARCH=$1

if [ "$ARCH" = "amd64" ]; then
  curl -L -o "bin/atlas-$ARCH" https://release.ariga.io/atlas/atlas-linux-amd64-latest
elif [ "$ARCH" = "arm64" ]; then
  curl -L -o "bin/atlas-$ARCH" https://release.ariga.io/atlas/atlas-linux-arm64-latest
fi

chmod +x "bin/atlas-$ARCH"