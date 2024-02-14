package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
	"warrencni/config"
	"warrencni/containers"
	"warrencni/network"

	v1 "k8s.io/api/core/v1"
)

func main() {
	argsWithoutProg := os.Args[1:]
	cfgFile := "defaultconfig.json"
	if len(argsWithoutProg) > 0 {
		cfgFile = argsWithoutProg[0]
	}

	config.LoadConfig(cfgFile)

	/*for srcIP, dstIP := range config.Cfg.RemoteNodes {
		//saddr, _ := netip.ParseAddr(getLocalIPv6())
		daddr, _ := netip.ParseAddr(dstIP.PublicIPv6)
		mask, _ := netip.ParseAddr(strings.Split(srcIP, "/")[0])

		fmt.Printf("Added address to ebpf map target %s to %s\n", mask.String(), daddr.String())

	}*/

	cri := (&containers.ContainerdRuntimeInterface{}).Init()

	containers.InitCgroups()
	network.InitContainerNetworking()

	network.SetupTunnel()

	for i := 0; i < config.Cfg.NumServices; i++ {
		fmt.Printf("Reading svc%d.json\n", i)
		jsonBytes, err := os.ReadFile(fmt.Sprintf("svc%d.json", i))
		if err != nil {
			fmt.Printf("Failed to read svc%d.json", i)
		}
		pod := &v1.Pod{}
		err = json.Unmarshal(jsonBytes, pod)
		if err != nil {
			fmt.Printf("Failed to parse svc%d.json", i)
		}
		cri.DeployPod(pod)
	}

	network.SetupRoutes(config.Cfg.RemoteNodes)

	for {
		time.Sleep(time.Second)
	}

}
