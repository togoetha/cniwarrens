#!/bin/bash -v

#if [[ -z "${4:-}" ]]; then
#  echo "Use: startcni.sh " 1>&2
#  exit 1
#fi

subnet=${1}
shift
mask=${1}
shift
bridgeip=${1}

echo
brctl addbr cni0
ip link set cni0 up

ip addr add $bridgeip/$mask dev cni0

ip6tables -t filter -A FORWARD -s $subnet/$mask -j ACCEPT
ip6tables -t filter -A FORWARD -d $subnet/$mask -j ACCEPT