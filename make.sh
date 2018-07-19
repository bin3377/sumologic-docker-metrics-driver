#!/bin/sh

name=sumologic/sumologic-docker-metrics-plugin

docker build -f Dockerfile -t rootfsimage .
id=$(docker create "$name" true)
rm -rf rootfs
mkdir rootfs
docker export "$id" | tar -x -C rootfs
docker rm -vf "$id"
rm -rf rootfs/proc rootfs/sys rootfs/go rootfs/dev

docker plugin disable "$name"
docker plugin rm -f "$name"
docker plugin create "$name" .
# docker plugin enable "$name"
# rm -rf rootfs
