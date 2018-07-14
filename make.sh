#!/bin/sh

name=sumologic/sumologic-docker-metrics-driver
docker build -f Dockerfile -t "$name" .

id=$(docker create "$name")

rm -rf rootfs
mkdir -p rootfs
docker export "$id" | tar -zxvf - -C rootfs
docker rm "$id"

rm -rf rootfs/proc rootfs/sys rootfs/go rootfs/etc rootfs/dev

docker plugin disable "$name"
docker plugin rm -f "$name"
docker plugin create "$name" .
docker plugin enable "$name"
# rm -rf rootfs
