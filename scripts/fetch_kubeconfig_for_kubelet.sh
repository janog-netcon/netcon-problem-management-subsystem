#!/bin/sh

set -e

NAMESPACE="netcon"
SERVICEACCOUNT_NAME="netcon-nclet"
SECRET_NAME="$(kubectl get -n "${NAMESPACE}" ServiceAccounts "${SERVICEACCOUNT_NAME}" -o "jsonpath={.secrets[0].name}")"
TOKEN="$(kubectl get -n "${NAMESPACE}" Secrets "${SECRET_NAME}" -o "jsonpath={.data.token}" | base64 -d)"

cat <<EOF
kind: Config
apiVersion: v1
current-context: netcon
clusters:
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