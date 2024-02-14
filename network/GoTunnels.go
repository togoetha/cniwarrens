package network

import (
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"
	"warrencni/config"
	"warrencni/utils"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
)

func SetupTunnel() {

	//cmd := fmt.Sprintf("ip addr add %s/24 dev %s", config.Cfg.LocalIP, config.Cfg.ListenDev)
	//utils.ExecCmdBash(cmd)

	if config.Cfg.UseWG {
		go func() {
			//SetupWGRoutes()
		}()
	} else if config.Cfg.UseXDP {
		go func() {
			MaintainXDPTunnels()
		}()
	} else {
		cfg := water.Config{
			DeviceType: water.TUN,
		}
		cfg.Name = config.Cfg.TunDev

		ifce, err := water.New(cfg)
		if err != nil {
			log.Fatal(err)
		}
		cmd := fmt.Sprintf("ip -6 addr add %s/112 dev %s\n", GetTunnelIfIPv6(), config.Cfg.TunDev)
		utils.ExecCmdBash(cmd)
		cmd = fmt.Sprintf("ip link set %s up\n", config.Cfg.TunDev)
		utils.ExecCmdBash(cmd)

		conn, err := getUDPConnection()
		if err != nil {
			fmt.Println(err)
		}

		go func() {
			remoteToTun(ifce, conn)
		}()

		go func() {
			tunToRemote(ifce, conn)
		}()
	}
}

func getLocalIPv6() string {
	hwIfIp, _ := utils.ExecCmdBash("ip -6 r g 2a02::1/128 | grep -oP 'src \\K\\S+'")
	hwIfIp = strings.Trim(hwIfIp, "\r\n")
	return hwIfIp
}

func getExternalIPv6() string {
	extIp, _ := utils.ExecCmdBash("dig -6 TXT +short o-o.myaddr.l.google.com @ns1.google.com")
	extIp = strings.Trim(extIp, "\"")
	return extIp
}

func getLocalIP() string {
	hwIfIp, _ := utils.ExecCmdBash("ip r g 1.1.1.1 | grep -oP 'src \\K\\S+'")
	return hwIfIp
}

/*func getTargetIP() (string, int) {
	targetIP := ""
	port := 0
	for _, ip := range config.Cfg.RemoteNodes {
		targetIP, port = ip.PublicIPv6, ip.Port
	}
	return targetIP, port
}*/

func ContainerCreated(dstIP string, vethDevice string, containerDevice string) {
	if config.Cfg.UseXDP {
		AddRTTInterfaceLink(dstIP, vethDevice, containerDevice)
	}
}

func getUDPConnection() (*net.PacketConn, error) {
	laddr := fmt.Sprintf("[%s]:%d", getLocalIPv6(), config.Cfg.TunnelPort)
	listener, err := net.ListenPacket("udp", laddr)

	if err != nil {
		return nil, err
	}
	fmt.Printf("Listening at %s\n", laddr)
	return &listener, nil
}

func tunToRemote(ifce *water.Interface, conn *net.PacketConn) {
	var data ethernet.Frame

	addressMap := make(map[netip.Prefix]*net.UDPAddr)
	for srcIP, dstIP := range config.Cfg.RemoteNodes {
		daddr := &net.UDPAddr{
			Port: dstIP.Port, //config.Cfg.TunnelPort,
			IP:   net.ParseIP(dstIP.PublicIPv6),
		}
		//strings.Split(srcIP, "/")
		saddr, _ := netip.ParseAddr(strings.Split(srcIP, "/")[0])
		prefix, _ := saddr.Prefix(112)
		addressMap[prefix] = daddr
		fmt.Printf("Added address map %s prefix %s to %s", saddr.String(), prefix.String(), daddr.String())
	}

	//var dst net.IP
	for {
		data.Resize(1500)
		n, err := ifce.Read([]byte(data))
		if err != nil {
			log.Fatal(err)
		}
		data = data[:n]
		var src, dst net.IP

		//thanks golang, for making me duplicate this dumbass code
		if data[0]/16 == 4 {
			//fmt.Println("tunToRemote: IPv4")
			packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
			layer := packet.Layer(layers.LayerTypeIPv4)
			ip := layer.(*layers.IPv4)
			_, dst = ip.SrcIP, ip.DstIP

		} else if data[0]/16 == 6 {
			//fmt.Println("tunToRemote: IPv6")
			packet := gopacket.NewPacket(data, layers.LayerTypeIPv6, gopacket.Default)
			layer := packet.Layer(layers.LayerTypeIPv6)
			ip := layer.(*layers.IPv6)
			_, dst = ip.SrcIP, ip.DstIP
		} else {
			fmt.Printf("tunToRemote: unknown IP version %d\n", data[0]/16)
		}

		addr, _ := netip.AddrFromSlice(dst)
		//reduce to /112
		pref, _ := addr.Prefix(112)
		if config.Cfg.Debug {
			fmt.Printf("tunToRemote: dst %s src %s pref%s\n", dst, src, pref)
		}

		(*conn).WriteTo(data, addressMap[pref])
		//fmt.Printf("tunToRemote: sent to %s\n", addressMap[pref])
	}
}

func remoteToTun(ifce *water.Interface, conn *net.PacketConn) {
	buff := make([]byte, 2048)

	//fmt.Printf("IP Addresses external %s hw if %s\n", getExternalIPv6(), getLocalIPv6())

	for {
		n, remoteaddr, err := (*conn).ReadFrom(buff)
		if err != nil {
			fmt.Printf("remoteToTun: error  %v", err)
			continue
		}
		if config.Cfg.Debug {
			fmt.Printf("remoteToTun: read packet from %v\n", remoteaddr)
		}
		data := buff[:n]

		//DEBUG STUFF, REMOVE LATER
		if config.Cfg.Debug {
			var src, dst net.IP

			//thanks golang, for making me duplicate this dumbass code
			if data[0]/16 == 4 {
				fmt.Println("remoteToTun: IPv4")
				packet := gopacket.NewPacket(data, layers.LayerTypeIPv4, gopacket.Default)
				layer := packet.Layer(layers.LayerTypeIPv4)
				ip := layer.(*layers.IPv4)
				src, dst = ip.SrcIP, ip.DstIP

			} else if data[0]/16 == 6 {
				fmt.Println("remoteToTun: IPv6")
				packet := gopacket.NewPacket(data, layers.LayerTypeIPv6, gopacket.Default)
				layer := packet.Layer(layers.LayerTypeIPv6)
				ip := layer.(*layers.IPv6)
				src, dst = ip.SrcIP, ip.DstIP
			} else {
				fmt.Printf("remoteToTun: unknown IP version %d\n", data[0]/16)
			}
			fmt.Printf("remoteToTun: dst %s src %s\n", dst, src)
		}

		ifce.Write(data)
	}
}
