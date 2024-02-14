#!/bin/bash -vx

if [[ -z "${5:-}" ]]; then
  echo "Use: setupcontainercni.sh containername pid cniif gwip subnetsize bandwidth latency" 1>&2
  exit 1
fi
#%s %d eth1 %s %d %s", netNs, pid, ip, subnetMask, gatewayIP
containername=${1}
shift
hostif=${1}
shift
cniif=${1}
shift
containerip=${1}
shift
subnetsize=${1}
shift
gwip=${1}

INSIDE_MAC=$(cat /sys/class/net/$cniif/address)
ip neigh add $containerip lladdr $INSIDE_MAC dev $hostif nud permanent

#link cniif, add it to the right namespace and add a route 
ip link set $cniif netns $containername
ip netns exec $containername ip link set $cniif up
ip netns exec $containername ip -6 addr add $containerip/$subnetsize dev $cniif
ip netns exec $containername ip -6 route replace default via $gwip dev $cniif 
ip -6 route add $containerip dev cni0

OUTSIDE_MAC=$(cat /sys/class/net/$hostif/address)
ip netns exec $containername ip neigh add $gwip lladdr $OUTSIDE_MAC dev $cniif nud permanent