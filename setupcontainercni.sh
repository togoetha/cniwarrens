#!/bin/bash -vx

if [[ -z "${4:-}" ]]; then
  echo "Use: setupcontainercni.sh containername pid cniif hostif" 1>&2
  exit 1
fi
#%s %d eth1 %s %d %s", netNs, pid, ip, subnetMask, gatewayIP
containername=${1}
shift
pid=${1}
shift
cniif=${1}
shift
hostif=${1}
#containerip=${1}
#shift
#subnetsize=${1}
#shift
#gwip=${1}

ip netns attach $containername $pid

#generate device name and create veth, linking it to container device
#rand=$(tr -dc 'A-F0-9' < /dev/urandom | head -c4)
#hostif="veth$rand"
ip link add $cniif type veth peer name $hostif 
ifconfig $cniif mtu 1420 up
ethtool -K $cniif tx off sg off tso off

#link $hostif to cni0
ip link set $hostif up 
ip link set $hostif master cni0 

#delete any stuff docker made first, we don't want that interfering
ip netns exec $containername ip link delete eth0
ip netns exec $containername ip link delete $cniif

#INSIDE_MAC=$(cat /sys/class/net/$cniif/address)
#ip neigh add $containerip lladdr $INSIDE_MAC dev $hostif nud permanent

#link cniif, add it to the right namespace and add a route 
#ip link set $cniif netns $containername
#ip netns exec $containername ip link set $cniif up
#ip netns exec $containername ip -6 addr add $containerip/$subnetsize dev $cniif
#ip netns exec $containername ip -6 route replace default via $gwip dev $cniif 
#ip -6 route add $containerip dev cni0

#OUTSIDE_MAC=$(cat /sys/class/net/$hostif/address)
#ip netns exec $containername ip neigh add $gwip lladdr $OUTSIDE_MAC dev $cniif nud permanent