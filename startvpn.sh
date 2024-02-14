mask=${1}
shift
tunip=${1}

echo
ip link add tun0 type dummy
ifconfig tun0 hw ether C8:D7:4A:4E:47:50
ip addr add $tunip/$mask dev tun0

ip link set tun0 up

