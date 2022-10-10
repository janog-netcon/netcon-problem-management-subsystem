# How to set up a development environment

This page describes how to set up a development environment like the following.

![./devenv.svg](./devenv.svg)

## Installing prerequisites

To set up a development environment, you need to install the following items:

* kind
* kubectl

**Golang**

All controllers in netcon-problem-management-subsystem are written in Golang. First, to develop controllers, you need to install Golang development environment.

```bash
wget https://go.dev/dl/go1.19.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.19.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
. ~/.bashrc
```

**Docker**

In order to run Kubernetes cluster on your machine and run nclet, you need to install Docker.

```bash
curl -L https://get.docker.com | sudo sh
```

If you'd like to execute `docker` without `sudo` command, you can add your user to `docker` group.

```bash
sudo usermod -aG docker "$(id -un)"
```

**kind**

[kind](https://kind.sigs.k8s.io/) is a handy tool to run Kubernetes clusters on your local machine.

```bash
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.16.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

**kubectl**

Of course, to communicate with Kubernetes, you need to install kubectl.

```bash
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin
```

## Deploying controller-manager

First, you need to set up a Kubernetes cluster to run controllers.

```bash
kind create cluster
```

Next, you need to install cert-manager to generate and inject self-signed certificates for controller-manager.

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
```

Then, you can build and install controller-manager with these commands.

```
make controller-manager-docker-build
make controller-manager-kind-push
```

Finally, you can deploy controller-manager with this command.

```bash
make deploy
```

You can check the status of controller-manager with `kubectl -n netcon get pods`.

```bash
$ kubectl -n netcon get pods 
NAME                                         READY   STATUS    RESTARTS   AGE
netcon-controller-manager-66db9dd4fb-kcbb6   2/2     Running   0          14m
```

## Deploying nclet

After deploying controller-manager successfully, you can install nclet with these command. Note that workers can communicate with Kubernetes controll plane.

```bash
kind get kubeconfig > kubeconfig
mkdir ./data && chmod 0777 ./data
docker run -d --name nclet \
    --privileged --net=host --ipc=host \
    -e KUBECONFIG=/etc/kubernetes/kubeconfig \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $(pwd)/kubeconfig:/etc/kubernetes/kubeconfig \
    -v $(pwd)/data:/data \
    netcon-pms-nclet:dev
```
