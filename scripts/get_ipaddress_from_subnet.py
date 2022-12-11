#!/usr/bin/env python3

import sys
import ipaddress

if len(sys.argv) != 3:
    print("Usage: %s SUBNET OFFSET" % sys.argv[0])
    sys.exit(1)

network = sys.argv[1]
offset = int(sys.argv[2])

print(ipaddress.ip_network(network)[offset])
