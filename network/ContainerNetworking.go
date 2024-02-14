package network

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"strings"
	"warrencni/config"
	"warrencni/utils"
)

var baseSubnetIP string

var subnetReservedIPs int
var maxSubnetIP int

var subnetMask int

var usedAddresses map[string]string

func InitContainerNetworking() {
	baseSubnetIP = initBaseIPv6()
	subnetReservedIPs = 16 //start at 16, we reserve 16 IP addresses for system interfaces i.e. cni, vpn, ..
	maxSubnetIP = int(math.Pow(2, 16))
	subnetMask = 112

	/*cmd := fmt.Sprintf("sh -x ./startvpn.sh %d %s", subnetMask, GetTunnelIfIPv6())
	output, err := utils.ExecCmdBash(cmd)
	fmt.Println(output)
	if err != nil {
		fmt.Println("Could not set up vpn0")
	}*/

	cmd := fmt.Sprintf("sh -x ./startcni.sh %s %d %s", fmt.Sprintf("%s:%d", baseSubnetIP, 1), subnetMask, GetCNIIfIPv6())
	output, err := utils.ExecCmdBash(cmd)
	fmt.Println(output)
	if err != nil {
		fmt.Println("Could not set up cni0")
	}

	usedAddresses = make(map[string]string)
}

func RequestIP(namespace string, pod string) (string, error) {
	freeIP := subnetReservedIPs
	podName := namespace + "_" + pod
	fullIP := getContainerIPv6(freeIP)
	_, taken := usedAddresses[fullIP]
	for taken {
		freeIP++
	}
	if freeIP < maxSubnetIP {
		usedAddresses[fullIP] = podName
		return fullIP, nil
	} else {
		return "", errors.New("Out of IP addresses")
	}
}

func FreeIP(namespace string, pod string) {
	var foundIp string = ""
	podName := namespace + "_" + pod
	for ip, cName := range usedAddresses {
		if cName == podName {
			foundIp = ip
		}
	}
	if foundIp != "" {
		delete(usedAddresses, foundIp)
	}
}

func SetupRoutes(ipRouteMap map[string]config.NodeInfo) {
	// ip route add $nodeip/$mask via $routeip dev $extif

	for fledgeIPMask, _ := range ipRouteMap {
		//routeDev := GetRouteDev(publicIP)
		routeDev := config.Cfg.TunDev

		fledgeIP := strings.Split(fledgeIPMask, "/")
		tunIP := GetTunnelIfIPv6()
		//cmd := fmt.Sprintf("sh -x ./addroute.sh %s %s %s %s", fledgeIP[0], fledgeIP[1], tunIP, routeDev)
		cmd := fmt.Sprintf("ip -6 route add %s/%s dev %s", fledgeIP[0], fledgeIP[1], routeDev)
		_, err := utils.ExecCmdBash(cmd)
		if err != nil {
			fmt.Printf("Failed to set up route %s to %s\n", tunIP, fledgeIP)
		}
	}
}

func GetSubnetMask() int {
	return subnetMask
}

func GetRouteDev(publicIP string) string {
	cmd := fmt.Sprintf("ip route get %s | grep -E -o '[0-9\\.]* dev [a-z0-9]*'", publicIP)
	route, err := utils.ExecCmdBash(cmd)
	if err != nil {
		fmt.Println("Failed to determine public dev")
	}
	routeDev := strings.Split(route, " ")[2]
	return routeDev
	//return config.Cfg.TunDev
}

func GetTunnelIfIPv6() string {
	return fmt.Sprintf("%s:%s", baseSubnetIP, "1")
}

func GetCNIIfIPv6() string {
	return fmt.Sprintf("%s:%s", baseSubnetIP, "2")
}

func getContainerIPv6(container int) string {
	suffix := container + 16 //leave 4^2 dedicated addresses
	return fmt.Sprintf("%s:%x", baseSubnetIP, suffix)
}

func initBaseIPv6() string {
	if config.Cfg.Debug {
		return config.Cfg.CNIPrefix
	} else {
		machineId, err := os.ReadFile("/etc/machine-id")
		if err != nil {
			fmt.Println("Couldn't read machine id")
		}
		hash := fnv.New64a()
		hash.Write(machineId)

		machineHash := fmt.Sprintf("%x", hash.Sum64())
		ipv6Base := fmt.Sprintf("fd53:7769:726c:%s:%s:%s:%s", machineHash[0:4], machineHash[4:8], machineHash[8:12], machineHash[12:16])

		return ipv6Base
	}
}
