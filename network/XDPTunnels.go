package network

import (
	"fmt"
	"net"
	"time"

	// #include <linux/in6.h>
	//"C"

	"io"
	"net/netip"
	"strings"
	"warrencni/config"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
)

type In6ipport struct {
	DAddr   [16]byte
	SAddr   [16]byte
	Port    uint32
	SMac    [6]uint8
	DMac    [6]uint8
	Ifindex uint32
}

type EthLnk struct {
	//DAddr   [16]byte
	SMac    [6]uint8
	DMac    [6]uint8
	Ifindex uint32
}

// var tunnelMap *ebpf.Map
var iflnkMap *ebpf.Map

func SetupXDPTunnel() ([]io.Closer, []link.Link) {
	ttrClosers, l1 := SetupXDPTTR()
	rttClosers, l2 := SetupXDPRTT()

	for _, closer := range rttClosers {
		ttrClosers = append(ttrClosers, closer)
	}
	return ttrClosers, []link.Link{l1, l2}
}

func MaintainXDPTunnels() {
	_, links := SetupXDPTunnel()
	for true {
		restart := false
		for _, link := range links {
			info, err := link.Info()
			if err != nil {
				fmt.Printf("Error getting link info, triggering XDP reset\n")
				restart = true
			}
			idx := info.XDP().Ifindex
			//fmt.Printf("Link attached to if %d\n", idx)
			if idx <= 0 {
				fmt.Printf("Invalid link idx, triggering XDP reset\n")
				restart = true
			}
		}
		if restart {
			//close all and restart
			/*for _, closer := range closers {
				closer.Close()
			}*/
			//closers, links = SetupXDPTunnel()
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func getPublicIface() (*net.Interface, error) {
	var ipr config.NodeInfo
	for key, _ := range config.Cfg.RemoteNodes {
		ipr = config.Cfg.RemoteNodes[key]
		break
	}

	ifaceName := strings.Trim(GetRouteDev(ipr.PublicIPv6), "\n")

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Printf("lookup network iface %q: %s\n", ifaceName, err)
	}
	return iface, err
}

func getCNIIface() (*net.Interface, error) {
	ifaceName := "cni0"
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		fmt.Printf("lookup network iface %q: %s\n", ifaceName, err)
	}
	return iface, err
}

// TODO create calling code in main, refactor so first the device is created (and mac assigned), then this is
// called so the map entry is created, and only then the containerDevice is moved to container namespace
// also change calling code/containercni script so veth pair is named from golang instead of randomly from script
func AddRTTInterfaceLink(dstIP string, vethDevice string, containerDevice string) {
	fmt.Printf("Adding interface link for %s from cni dev %s to veth %s\n", dstIP, containerDevice, vethDevice)
	cniIface, err := net.InterfaceByName(containerDevice)
	if err != nil {
		fmt.Printf("Couldn't fetch device by name %s\n", containerDevice)
	}
	vethIface, err := net.InterfaceByName(vethDevice)
	if err != nil {
		fmt.Printf("Couldn't fetch device by name %s\n", vethDevice)
	}

	dAddr := [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//fucks sake golang REALLY
	for i := 0; i < 6; i++ {
		dAddr[i] = uint8(cniIface.HardwareAddr[i])
	}

	sAddr := [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//fucks sake golang REALLY
	for i := 0; i < 6; i++ {
		sAddr[i] = uint8(vethIface.HardwareAddr[i])
	}

	fmt.Printf("Mac addresses from %v to %v\n", sAddr, dAddr)

	netAddr, _ := netip.ParseAddr(dstIP)
	err = iflnkMap.Put(netAddr.As16(),
		EthLnk{
			//DAddr:   daddr.As16(), //target physical address
			SMac:    sAddr, //source machine phys mac
			DMac:    dAddr, //target machine phys mac
			Ifindex: uint32(vethIface.Index),
		})
	if err != nil {
		fmt.Printf("Failed to add to eth info map %s\n", err.Error())
	} else {
		fmt.Printf("Added address to eth info map from if %s index %d cni0 %v to %s %v ipv6 %v\n", vethDevice, cniIface.Index, sAddr, containerDevice, dAddr, dstIP)
	}

	var key [16]byte
	var val In6ipport

	iter := iflnkMap.Iterate()
	for iter.Next(&key, &val) {
		fmt.Printf("Map target cni [%v] bytes %v tunnel from local [%v] to [%v]:%d through if %d\n", netip.AddrFrom16(key), key, netip.AddrFrom16(val.SAddr), netip.AddrFrom16(val.DAddr), val.Port, val.Ifindex)
	}
}

func SetupXDPRTT() ([]io.Closer, link.Link) {
	iface, err := getPublicIface()
	if err != nil {
		return []io.Closer{}, &link.RawLink{}
	}

	objs := BpfObjectsRTT{}
	if err := loadBpfObjectsRTT(&objs, nil); err != nil {
		fmt.Printf("loading objects: %s\n", err)
	}

	l, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpProgFunc,
		Interface: iface.Index,
		Flags:     link.XDPGenericMode,
	})
	if err != nil {
		fmt.Printf("could not attach XDP program: %s", err)
	}

	iflnkMap = objs.XdpIfLnkMap
	fmt.Printf("Attached RTT XDP program to iface %q (index %d)\n", iface.Name, iface.Index)

	return []io.Closer{l, &objs}, l
}

func SetupXDPTTR() ([]io.Closer, link.Link) {
	iface, err := getCNIIface()
	if err != nil {
		return []io.Closer{}, &link.RawLink{}
	}

	// Load pre-compiled programs into the kernel.
	objs := BpfObjectsTTR{}
	if err := loadBpfObjectsTTR(&objs, nil); err != nil {
		panic(err)
	}

	// Attach the program.
	l2, err := link.AttachXDP(link.XDPOptions{
		Program:   objs.XdpProgFunc,
		Interface: iface.Index,
		Flags:     link.XDPGenericMode,
	})
	if err != nil {
		fmt.Printf("could not attach XDP program: %s", err)
	}

	pubIface, err := getPublicIface()
	hwAddr := [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	//fucks sake golang REALLY
	for i := 0; i < 6; i++ {
		hwAddr[i] = uint8(pubIface.HardwareAddr[i])
	}

	fmt.Printf("Attached TTR XDP program to iface %q (index %d mac %s bytes %v)\n", iface.Name, iface.Index, iface.HardwareAddr.String(), hwAddr)

	for srcIP, dstIP := range config.Cfg.RemoteNodes {
		saddr, _ := netip.ParseAddr(getLocalIPv6())
		daddr, _ := netip.ParseAddr(dstIP.PublicIPv6)
		mask, _ := netip.ParseAddr(strings.Split(srcIP, "/")[0])
		err = objs.XdpTunnelMap.Put(mask.As16(),
			In6ipport{
				DAddr:   daddr.As16(), //target physical address
				SAddr:   saddr.As16(), //this node's physical address
				Port:    uint32(dstIP.Port),
				SMac:    hwAddr,          //source machine phys mac
				DMac:    dstIP.PublicMac, //target machine phys mac
				Ifindex: uint32(pubIface.Index),
			})
		if err != nil {
			fmt.Printf("Failed to add to tunnel map %s\n", err.Error())
		} else {
			fmt.Printf("Added address to tunnel map target %s from this node %s:[%v] to %s:[%v]\n", mask.String(), saddr.String(), hwAddr, daddr.String(), dstIP.PublicMac)
		}
	}
	//tunnelMap = objs.XdpTunnelMap

	var key [16]byte
	var val In6ipport

	iter := objs.XdpTunnelMap.Iterate()
	for iter.Next(&key, &val) {
		fmt.Printf("Map target cni [%v] bytes %v tunnel from local [%v] to [%v]:%d through if %d\n", netip.AddrFrom16(key), key, netip.AddrFrom16(val.SAddr), netip.AddrFrom16(val.DAddr), val.Port, val.Ifindex)
	}

	go func() {
		for true {
			//info, _ := l2.Info()
			//fmt.Printf("TTR Attached to %d", info.XDP().Ifindex)
			time.Sleep(time.Second)
		}
	}()

	return []io.Closer{l2, &objs}, l2
}

func loadBpfTTR() (*ebpf.CollectionSpec, error) {
	spec, err := ebpf.LoadCollectionSpec("xdp_tun_to_remote.o")
	if err != nil {
		return nil, fmt.Errorf("can't load bpf: %w", err)
	}

	return spec, err
}

func loadBpfObjectsTTR(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadBpfTTR()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

func loadBpfRTT() (*ebpf.CollectionSpec, error) {
	spec, err := ebpf.LoadCollectionSpec("xdp_remote_to_tun.o")
	if err != nil {
		return nil, fmt.Errorf("can't load bpf: %w", err)
	}

	return spec, err
}

func loadBpfObjectsRTT(obj interface{}, opts *ebpf.CollectionOptions) error {
	spec, err := loadBpfRTT()
	if err != nil {
		return err
	}

	return spec.LoadAndAssign(obj, opts)
}

type BpfObjectsTTR struct {
	BpfProgramsTTR
	BpfMapsTTR
}

type BpfObjectsRTT struct {
	BpfProgramsRTT
	BpfMapsRTT
}

type BpfMapsTTR struct {
	XdpTunnelMap *ebpf.Map `ebpf:"xdp_tunnel_map"`
}

type BpfProgramsTTR struct {
	XdpProgFunc *ebpf.Program `ebpf:"xdp_prog_ttr"`
	//XdpDummyProgFunc *ebpf.Program `ebpf:"xdp_pass_func"`
}

type BpfMapsRTT struct {
	XdpIfLnkMap *ebpf.Map `ebpf:"xdp_if_map"`
}

type BpfProgramsRTT struct {
	XdpProgFunc *ebpf.Program `ebpf:"xdp_prog_rtt"`
}

func (p *BpfProgramsTTR) Close() error {
	return _BpfClose(
		p.XdpProgFunc,
	)
}

func (p *BpfProgramsRTT) Close() error {
	return _BpfClose(
		p.XdpProgFunc,
	)
}

func _BpfClose(closers ...io.Closer) error {
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}
