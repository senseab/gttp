#!/bin/bash
export GOOS=linux
export IMAGE=tonychee7000/gttp
export TAG=`date +%Y%m%d`-`git log|head -1|awk '{print $2}'|cut -c -8`
go build -ldflags "-w -s"
upx -9 gttp
docker build -t $IMAGE:$TAG .
docker tag $IMAGE:$TAG $IMAGE:latest
docker push $IMAGE:$TAG 
docker push $IMAGE:latest