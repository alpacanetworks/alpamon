#!/bin/bash

docker build -t alpamon:ubuntu-18.04 -f tests/ubuntu/18.04/Dockerfile .
docker build -t alpamon:ubuntu-20.04 -f tests/ubuntu/20.04/Dockerfile .
docker build -t alpamon:ubuntu-22.04 -f tests/ubuntu/22.04/Dockerfile .

docker build -t alpamon:debian-10 -f tests/debian/10/Dockerfile .
docker build -t alpamon:debian-11 -f tests/debian/11/Dockerfile .

docker build -t alpamon:centos-7 -f tests/centos/7/Dockerfile .
docker build -t alpamon:redhat-8 -f tests/redhat/8/Dockerfile .
docker build -t alpamon:redhat-9 -f tests/redhat/9/Dockerfile .