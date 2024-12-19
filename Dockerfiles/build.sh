#!/bin/bash

docker build -t alpamon:debian-10 -f Dockerfiles/debian/10/Dockerfile .
docker build -t alpamon:debian-11 -f Dockerfiles/debian/11/Dockerfile .

docker build -t alpamon:ubuntu-18.04 -f Dockerfiles/ubuntu/18.04/Dockerfile .
docker build -t alpamon:ubuntu-20.04 -f Dockerfiles/ubuntu/20.04/Dockerfile .
docker build -t alpamon:ubuntu-22.04 -f Dockerfiles/ubuntu/22.04/Dockerfile .

docker build -t alpamon:redhat-8 -f Dockerfiles/redhat/8/Dockerfile .
docker build -t alpamon:redhat-9 -f Dockerfiles/redhat/9/Dockerfile .

docker build -t alpamon:centos-7 -f Dockerfiles/centos/7/Dockerfile .