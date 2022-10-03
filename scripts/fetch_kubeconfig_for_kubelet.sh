#!/bin/sh

set -ex

NAMESPACE="netcon"
SERVICEACCOUNT_NAME="nclet"
SECRET_NAME="$(kubectl get -n "${NAMESPACE}" get ServiceAccounts "${SERVICEACCOUNT_NAME}" -o "jsonpath=.status.secrets[0]")"
TOKEN="$(kubectl get -n "${NAMESPACE}" get Secrets "${SECRET_NAME}" -o "jsonpath=.data.token" | base64 -d)"

echo <<EOF
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
    token: ${TOKEN}
EOF