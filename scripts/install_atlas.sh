#!/bin/bash

mkdir -p bin

ARCH=${GOARCH}

if [ "$ARCH" = "amd64" ]; then
  curl -L -o bin/atlas https://release.ariga.io/atlas/atlas-community-linux-amd64-latest
elif [ "$ARCH" = "arm64" ]; then
  curl -L -o bin/atlas https://release.ariga.io/atlas/atlas-community-linux-arm64-latest
fi

chmod +x bin/atlas
