#!/bin/sh

docker rm -f nclet

docker run -d --name nclet \
    --privileged --pid=host --net=host --ipc=host \
    -e KUBECONFIG=/etc/kubernetes/kubeconfig \
    -v /proc:/proc \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $(pwd)/kubeconfig:/etc/kubernetes/kubeconfig \
    -v /data:/data \
    netcon-pms-nclet:dev -admin-password "netcon-pms"

