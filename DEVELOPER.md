## 
```
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.9.1/cert-manager.yaml
```

## How to deploy nclet (netcon-let)?

At first, we need to create ServiceAccount for nclet.

```
---
apiVersion: v1
kind: ServiceAccount
metadata:
  namespace: netcon
  name: nclet
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  namespace: netcon
  name: system:nclet
rules:
- apiGroups: ["netcon.janog.gr.jp"]
  resources: ["*"]
  verbs: ["get", "list", "watch", "create", "delete", "patch", "update"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  namespace: netcon
  name: system:nclet:nclet
subjects:
- kind: ServiceAccount
  namespace: netcon
  name: nclet
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:nclet
```

```
$ kubectl -n netcon get secret nclet-token-4gqlr -o yaml
apiVersion: v1
data:
  token: {{ b64encode(token) }}
kind: Secret
metadata:
  name: nclet-token-4gqlr
  namespace: netcon
type: kubernetes.io/service-account-token
```

```
kind: Config
apiVersion: v1
current-context: netcon
clusters
- name: netcon-cplane
  cluster:
    server: {{ server_address }}
contexts:
- name: netcon
  context:
    cluster: netcon-cplane
    user: nclet
users:
- name: nclet
  user:
    token: {{ token }}
```

```
$ ls -a
.  ..  kubeconfig
$ docker run -d \
    --name nclet \
    --net host \
    -e KUBECONFIG=/etc/kubernetes/kubeconfig \
    -v $(pwd)/kubeconfig:/etc/kubernetes/kubeconfig \
    proelbtn/netcon-pms-nclet:dev
```