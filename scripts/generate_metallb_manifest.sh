#!/bin/sh

# This script generate MetalLB configuration manifest automatically

set -e

cd $(dirname $0)

# The network name kind workers connect to
DOCKER_KIND_NETWORK_NAME="${DOCKER_KIND_NETWORK_NAME:-kind}"

LB_START_OFFSET="${LB_START_OFFSET:-200}"
LB_END_OFFSET="${LB_END_OFFSET:-250}"

NETWORK="$(docker network inspect -f '{{(index .IPAM.Config 0).Subnet}}' "${DOCKER_KIND_NETWORK_NAME}")"

LB_START="$(./get_ipaddress_from_subnet.py $NETWORK $LB_START_OFFSET)"
LB_END="$(./get_ipaddress_from_subnet.py $NETWORK $LB_END_OFFSET)"

cat <<EOF
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default
  namespace: metallb-system
spec:
  addresses:
  - ${LB_START}-${LB_END}
---
apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default
  namespace: metallb-system
EOF
