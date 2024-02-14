package containers

import (
	"fmt"
	"strings"
	"warrencni/network"
	"warrencni/utils"

	v1 "k8s.io/api/core/v1"
)

func GetNetNs(namespace string, pod string) string {
	return namespace + "-" + pod
}

func BindNetNamespace(namespace string, pod string, pid int) string {
	netNs := GetNetNs(namespace, pod)
	gatewayIP := network.GetCNIIfIPv6()
	ip, _ := network.RequestIP(namespace, pod)

	vethName := fmt.Sprintf("veth%s", strings.Split(ip, ":")[7])
	//This used to set up net namespace + pid IP address and some routing shit
	//now the functionality is split, this sets up the devices and assigns IP address, with a golang generated veth name
	cmd := fmt.Sprintf("sh -x ./setupcontainercni.sh %s %d eth1 %s", netNs, pid, vethName) //ip, network.GetSubnetMask(), gatewayIP)
	utils.ExecCmdBash(cmd)

	//notify the tunnels
	network.ContainerCreated(ip, vethName, "eth1")

	//and move to new namespace
	cmd = fmt.Sprintf("sh -x ./movecontainernamespace.sh %s %s eth1 %s %d %s", netNs, vethName, ip, network.GetSubnetMask(), gatewayIP)
	utils.ExecCmdBash(cmd)
	return ip
}

func GetNetworkNamespace(namespace string, pod *v1.Pod) string {
	nsName := namespace + "-" + pod.ObjectMeta.Name
	//cmd := fmt.Sprintf("ip netns add %s", nsName)
	//ExecCmdBash(cmd)

	nsPath := fmt.Sprintf("/var/run/netns/%s", nsName)
	return nsPath
}

func RemoveNetNamespace(namespace string, pod string) {
	netNs := GetNetNs(namespace, pod)
	cmd := fmt.Sprintf("sh -x ./shutdowncontainercni.sh %s eth1", netNs)
	utils.ExecCmdBash(cmd)
}
