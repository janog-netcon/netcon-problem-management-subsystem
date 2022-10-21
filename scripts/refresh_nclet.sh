#!/bin/sh

docker rm -f nclet

docker run -d --name nclet \
    --privileged --net=host --ipc=host \
    -e KUBECONFIG=/etc/kubernetes/kubeconfig \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $(pwd)/kubeconfig:/etc/kubernetes/kubeconfig \
    -v $(pwd)/data:/data \
    netcon-pms-nclet:dev
