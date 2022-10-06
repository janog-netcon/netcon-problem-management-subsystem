## Development Environment

### Golang

```bash
wget https://go.dev/dl/go1.19.2.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.19.2.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
. ~/.bashrc
```

## Docker

```bash
curl -L get.docker.com | sudo sh
sudo usermod -aG docker "$(id -un)"
```

### kind

```bash
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.16.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

### kubectl

```bash
curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
chmod +x kubectl && sudo mv kubectl /usr/local/bin
```

```bash
kind create cluster
```

## Deploy

### Prerequisite

* Kubernetes cluster
  1. This cluster needs to be accessible from Worker nodes
    * nclet needs to communicate with this cluster to work successfully
  2. Cert-manager needs to be installed on this cluster
    * it is needed to issue self-signed certificates and inject them to some manifests
  3. Optionally, ArgoCD may be installed on this cluster if needed
    * it helps us manage Kubernetes manifests
* Worker nodes
  1. Docker needs to be installed on Worker nodes
    * nclet will run as Docker container

You can install cert-manager with the following command:

```
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
```

### central components

You can install central components just by executing `make deploy` on the project root.

### nclet

Before deploying nclet to Worker node, you need to create kubeconfig file for nclet. You can create it with `./scripts/fetch_kubeconfig_for_kubelet.sh`.

```
$ ./scripts/fetch_kubeconfig_for_kubelet.sh
```

After creating kubeconfig, you can deploy nclet with the following commands. Note that they expect to be executed on Worker node.

```
$ ls -a
.  ..  kubeconfig
$ docker run -d \
    --name nclet \
    --privileged \
    --net host \
    --ipc host \
    -e KUBECONFIG=/etc/kubernetes/kubeconfig \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v $(pwd)/kubeconfig:/etc/kubernetes/kubeconfig \
    -v $(pwd)/data:/data \
    harbor.linecorp.com/jp26081/netcon-pms-nclet:dev
```
